package fitness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

const tokenBudgetLimit = 300_000

// LLMClient — автономный I/O-объект для OpenAI-совместимого LLM-провайдера.
type LLMClient struct {
	provider string
	baseURL  string
	model    string
	http     *http.Client
}

// NewLLMClient создаёт LLMClient с параметрами подключения.
// Ключ API читается из env при каждом вызове Simulate.
func NewLLMClient(provider, baseURL, model string) LLMClient {
	if provider == "" {
		provider = "anthropic"
	}
	if model == "" {
		if provider == "anthropic" {
			model = "claude-sonnet-4-6"
		} else {
			model = "gpt-4o"
		}
	}
	if provider == "anthropic" && baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	return LLMClient{
		provider: provider,
		baseURL:  baseURL,
		model:    model,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

// Simulate запускает LLM-оценку для каждого промпта из набора.
// Маппит HTTP-ошибки провайдера в доменные: 429→ErrLLMRateLimited,
// 5xx/сеть→ErrLLMUnavailable, превышение токенов→ErrLLMBudgetExceeded.
func (c LLMClient) Simulate(set domain.JTBDPromptSet) ([]domain.LLMVerdict, error) {
	envVar := "ANTHROPIC_API_KEY"
	if c.provider == "openai" {
		envVar = "OPENAI_API_KEY"
	}
	key := os.Getenv(envVar)
	if key == "" {
		return nil, fmt.Errorf("%w: %s не задан", domain.ErrLLMUnavailable, envVar)
	}

	verdicts := make([]domain.LLMVerdict, 0, len(set.Prompts()))
	for i, p := range set.Prompts() {
		if i > 0 {
			time.Sleep(10 * time.Second)
		}
		v, err := c.call(p, key)
		if err != nil {
			return nil, err
		}
		verdicts = append(verdicts, v)
	}
	return verdicts, nil
}

func (c LLMClient) call(p domain.JTBDPrompt, key string) (domain.LLMVerdict, error) {
	body := chatRequest{
		Model:     c.model,
		Messages:  []chatMessage{{Role: "user", Content: p.Text()}},
		MaxTokens: p.Budget(),
		ResponseFormat: map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "verdict",
				"strict": true,
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"status": map[string]any{"type": "string", "enum": []string{"PASS", "FAIL", "PARTIAL"}},
						"score":  map[string]any{"type": "integer"},
						"gaps":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					},
					"required":             []string{"status", "score", "gaps"},
					"additionalProperties": false,
				},
			},
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return domain.LLMVerdict{}, fmt.Errorf("%w: marshal: %v", domain.ErrLLMUnavailable, err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return domain.LLMVerdict{}, fmt.Errorf("%w: request: %v", domain.ErrLLMUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := c.http.Do(req)
	if err != nil {
		return domain.LLMVerdict{}, fmt.Errorf("%w: %v", domain.ErrLLMUnavailable, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return domain.LLMVerdict{}, fmt.Errorf("%w", domain.ErrLLMRateLimited)
	case http.StatusOK:
		// обрабатывается ниже
	default:
		return domain.LLMVerdict{}, fmt.Errorf("%w: HTTP %d", domain.ErrLLMUnavailable, resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return domain.LLMVerdict{}, fmt.Errorf("%w: decode: %v", domain.ErrLLMUnavailable, err)
	}

	if chatResp.Usage.TotalTokens > tokenBudgetLimit {
		return domain.LLMVerdict{}, fmt.Errorf("%w: usage %d > %d", domain.ErrLLMBudgetExceeded, chatResp.Usage.TotalTokens, tokenBudgetLimit)
	}

	if len(chatResp.Choices) == 0 {
		return domain.LLMVerdict{}, fmt.Errorf("%w: пустой ответ", domain.ErrLLMUnavailable)
	}

	raw := extractJSON(chatResp.Choices[0].Message.Content)
	var v verdictJSON
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return domain.LLMVerdict{}, fmt.Errorf("%w: parse verdict: %v", domain.ErrLLMUnavailable, err)
	}

	return domain.LLMVerdict{
		Consumer:  p.Consumer(),
		RawStatus: v.Status,
		RawScore:  v.Score,
		RawGaps:   v.Gaps,
	}, nil
}

// ── JSON-структуры OpenAI chat completions ───────────────────────────────────

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	MaxTokens      int            `json:"max_tokens"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

type verdictJSON struct {
	Status string   `json:"status"`
	Score  int      `json:"score"`
	Gaps   []string `json:"gaps"`
}

// extractJSON вырезает первый JSON-объект из строки ответа модели,
// игнорируя markdown-обёртки и любой текст вокруг.
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end < start {
		return s
	}
	return s[start : end+1]
}
