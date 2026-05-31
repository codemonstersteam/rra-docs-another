package fitness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// Бэкофф по Retry-After: база/потолок паузы между повторами transient-отказа.
const (
	retryBackoffBase = 1 * time.Second
	retryBackoffCap  = 30 * time.Second
)

// LLMClient — автономный I/O-объект для OpenAI-совместимого LLM-провайдера.
// Все параметры подключения (baseURL, model, ключ) приходят из валидированного
// domain.LLMConfig — клиент ничего не хардкодит и не резолвит сам (резолвинг —
// в domain.NewLLMConfig, ADR 0003).
type LLMClient struct {
	baseURL     string
	model       string
	apiKey      string
	callDelayMs int
	tokenBudget int
	maxRetries  int
	http        *http.Client
}

// NewLLMClient создаёт LLMClient из готового LLMConfig и операционных параметров.
// llmCfg — валидированное подключение (baseURL уже с нужным префиксом, ключ из env).
// callDelayMs — пауза между последовательными вызовами (0 = без паузы).
// tokenBudget — защитный лимит токенов на вызов (skill http-io); 0 → 300000.
// maxRetries — повторы на 429 с бэкоффом по Retry-After (0 = без повтора).
func NewLLMClient(llmCfg domain.LLMConfig, callDelayMs, tokenBudget, maxRetries int) LLMClient {
	if tokenBudget <= 0 {
		tokenBudget = 300_000
	}
	return LLMClient{
		baseURL:     llmCfg.BaseURL(),
		model:       llmCfg.Model(),
		apiKey:      llmCfg.APIKey(),
		callDelayMs: callDelayMs,
		tokenBudget: tokenBudget,
		maxRetries:  maxRetries,
		http:        &http.Client{Timeout: 30 * time.Second},
	}
}

// Ask запускает LLM-оценку для каждого промпта из набора.
// Маппит HTTP-ошибки провайдера в доменные: 429→ErrLLMRateLimited,
// 5xx/сеть→ErrLLMUnavailable, превышение токенов→ErrLLMBudgetExceeded.
func (c LLMClient) Ask(set domain.JTBDPromptSet) ([]domain.LLMVerdict, error) {
	key := c.apiKey

	// Пре-флайт payload-бюджета: не отправляем заведомо лишний контекст
	// (skill http-io → «Бюджет payload»). Защита ДО сетевого вызова.
	if est := promptSetTokens(set); overTokenBudget(est, c.tokenBudget) {
		return nil, fmt.Errorf("%w: оценка входа %d > %d (до вызова)", domain.ErrLLMBudgetExceeded, est, c.tokenBudget)
	}

	verdicts := make([]domain.LLMVerdict, 0, len(set.Prompts()))
	for i, p := range set.Prompts() {
		if i > 0 && c.callDelayMs > 0 {
			time.Sleep(time.Duration(c.callDelayMs) * time.Millisecond)
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

	// transient 429 → бэкофф по Retry-After и повтор (skill http-io → «Пацинг»).
	// maxRetries=0 (дефолт) → поведение прежнее: первый 429 сразу даёт sentinel.
	var resp *http.Response
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequest(http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(data))
		if err != nil {
			return domain.LLMVerdict{}, fmt.Errorf("%w: request: %v", domain.ErrLLMUnavailable, err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+key)

		resp, err = c.http.Do(req)
		if err != nil {
			return domain.LLMVerdict{}, fmt.Errorf("%w: %v", domain.ErrLLMUnavailable, err)
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < c.maxRetries {
			wait := retryWait(resp.Header.Get("Retry-After"), attempt, retryBackoffBase, retryBackoffCap)
			resp.Body.Close()
			time.Sleep(wait)
			continue
		}
		break
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

	if overTokenBudget(chatResp.Usage.TotalTokens, c.tokenBudget) {
		return domain.LLMVerdict{}, fmt.Errorf("%w: usage %d > %d", domain.ErrLLMBudgetExceeded, chatResp.Usage.TotalTokens, c.tokenBudget)
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
