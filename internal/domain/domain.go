// Package domain содержит типы и конструкторы доменной модели rra-docs-another.
// Валидируемые структуры имеют неэкспортируемые поля и создаются конструктором.
// Остальные — плоские DTO (публичные поля).
package domain

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed defaults/config.yaml
var defaultConfigYAML []byte

// ── Доменные sentinel-ошибки ────────────────────────────────────────────────

var (
	ErrPathNotFound      = errors.New("path_not_found")
	ErrReadError         = errors.New("read_error")
	ErrConfigInvalid     = errors.New("config_invalid")
	ErrToolMissing       = errors.New("tool_missing")
	ErrToolFailed        = errors.New("tool_failed")
	ErrLLMRateLimited    = errors.New("llm_rate_limited")
	ErrLLMUnavailable    = errors.New("llm_unavailable")
	ErrLLMBudgetExceeded = errors.New("llm_budget_exceeded")
)

// ── Невалидированный вход ────────────────────────────────────────────────────

// Request — плоский DTO из ингресс-адаптера.
type Request struct {
	Command     string
	Path        string
	Format      string
	Out         string
	ConfigPath  string
	LLMProvider string
	LLMBaseURL  string
	LLMModel    string
	UpTo        string
	Semantic    bool
}

// ── Валидируемые доменные структуры ─────────────────────────────────────────

// AuditTarget — валидированный корень репозитория (неэкспортируемые поля).
type AuditTarget struct {
	root   string
	commit string
}

func (t AuditTarget) Root() string   { return t.root }
func (t AuditTarget) Commit() string { return t.commit }

// NewAuditTarget валидирует путь и создаёт AuditTarget.
// Failure: ErrPathNotFound (нет пути / не директория), ErrReadError (нет прав).
func NewAuditTarget(req Request) (AuditTarget, error) {
	abs, err := filepath.Abs(req.Path)
	if err != nil {
		return AuditTarget{}, fmt.Errorf("%w: %s", ErrPathNotFound, req.Path)
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return AuditTarget{}, fmt.Errorf("%w: %s", ErrPathNotFound, abs)
		}
		return AuditTarget{}, fmt.Errorf("%w: %s", ErrReadError, abs)
	}
	if !info.IsDir() {
		return AuditTarget{}, fmt.Errorf("%w: %s не директория", ErrPathNotFound, abs)
	}
	if _, err := os.ReadDir(abs); err != nil {
		return AuditTarget{}, fmt.Errorf("%w: %s", ErrReadError, abs)
	}
	commit := headCommit(abs)
	return AuditTarget{root: abs, commit: commit}, nil
}

func headCommit(root string) string {
	data, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	if err != nil {
		return ""
	}
	head := string(data)
	if len(head) > 0 {
		return head[:min(len(head), 40)]
	}
	return ""
}

// ── Config ───────────────────────────────────────────────────────────────────

// configYAML — структура YAML-конфига для парсинга.
type configYAML struct {
	LLM struct {
		Provider     string `yaml:"provider"`
		Model        string `yaml:"model"`
		APIKeyEnv    string `yaml:"api_key_env"`
		BaseURL      string `yaml:"base_url"`
		CallDelayMs  int    `yaml:"call_delay_ms"`
	} `yaml:"llm"`
	Docs       []string          `yaml:"docs"`
	Prompts    map[string]string `yaml:"prompts"`
	Thresholds struct {
		DriftDays      int `yaml:"drift_days"`
		ReadabilityMin int `yaml:"readability_min"`
	} `yaml:"thresholds"`
}

// Config — валидированный проектный конфиг (неэкспортируемые поля).
type Config struct {
	driftThresholdDays int
	readabilityMin     int
	llmPrompts         map[string]string
	docs               []string
	llmCallDelayMs     int
}

func (c Config) DriftThresholdDays() int { return c.driftThresholdDays }
func (c Config) ReadabilityMin() int     { return c.readabilityMin }

// Docs возвращает список doc-файлов для проверки (относительные пути от корня репо).
func (c Config) Docs() []string { return c.docs }

// LLMCallDelayMs возвращает задержку между последовательными LLM-вызовами (мс).
// 0 = без задержки (дефолт для тестов). Для реального API рекомендуется 10000.
func (c Config) LLMCallDelayMs() int { return c.llmCallDelayMs }

// LLMPrompt возвращает промпт для роли (maintainer|consumer|manager|agent).
func (c Config) LLMPrompt(role string) string {
	if c.llmPrompts == nil {
		return ""
	}
	return c.llmPrompts[role]
}

// NewConfig валидирует и создаёт Config.
// Если ConfigPath пуст — берётся встроенный дефолт (go:embed).
// Failure: ErrConfigInvalid.
func NewConfig(req Request) (Config, error) {
	if req.ConfigPath == "" {
		return parseConfigYAML(defaultConfigYAML)
	}
	data, err := os.ReadFile(req.ConfigPath)
	if err != nil {
		return Config{}, fmt.Errorf("%w: %s", ErrConfigInvalid, err)
	}
	return parseConfigYAML(data)
}

func parseConfigYAML(data []byte) (Config, error) {
	var raw configYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("%w: %v", ErrConfigInvalid, err)
	}
	dt := raw.Thresholds.DriftDays
	if dt == 0 {
		dt = 90
	}
	rm := raw.Thresholds.ReadabilityMin
	if rm == 0 {
		rm = 50
	}
	return Config{
		driftThresholdDays: dt,
		readabilityMin:     rm,
		llmPrompts:         raw.Prompts,
		docs:               raw.Docs,
		llmCallDelayMs:     raw.LLM.CallDelayMs,
	}, nil
}

// ── LLMConfig ────────────────────────────────────────────────────────────────

// LLMConfig — валидированная конфигурация LLM (неэкспортируемые поля).
type LLMConfig struct {
	provider string
	baseURL  string
	model    string
	apiKey   string
}

func (c LLMConfig) Provider() string { return c.provider }
func (c LLMConfig) BaseURL() string  { return c.baseURL }
func (c LLMConfig) Model() string    { return c.model }
func (c LLMConfig) APIKey() string   { return c.apiKey }

// NewLLMConfig валидирует LLM-подключение и создаёт LLMConfig.
// Антецедент: provider ∈ {anthropic,openai}; для openai base_url непустой;
// ключ присутствует в env (ANTHROPIC_API_KEY | OPENAI_API_KEY).
// Failure: ErrLLMUnavailable.
func NewLLMConfig(req Request) (LLMConfig, error) {
	provider := req.LLMProvider
	if provider == "" {
		provider = "anthropic"
	}
	if provider != "anthropic" && provider != "openai" {
		return LLMConfig{}, fmt.Errorf("%w: провайдер %q неизвестен", ErrLLMUnavailable, provider)
	}

	baseURL := req.LLMBaseURL
	if provider == "openai" && baseURL == "" {
		return LLMConfig{}, fmt.Errorf("%w: openai требует --llm-base-url", ErrLLMUnavailable)
	}
	if provider == "anthropic" && baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	envVar := "ANTHROPIC_API_KEY"
	if provider == "openai" {
		envVar = "OPENAI_API_KEY"
	}
	key := os.Getenv(envVar)
	if key == "" {
		return LLMConfig{}, fmt.Errorf("%w: переменная %s не задана", ErrLLMUnavailable, envVar)
	}

	model := req.LLMModel
	if model == "" {
		if provider == "anthropic" {
			model = "claude-sonnet-4-6"
		} else {
			model = "gpt-4o"
		}
	}

	return LLMConfig{
		provider: provider,
		baseURL:  baseURL,
		model:    model,
		apiKey:   key,
	}, nil
}

// ── JTBDPrompt / JTBDPromptSet / LLMVerdict ─────────────────────────────────

// JTBDPrompt — промпт для одной JTBD-роли (неэкспортируемые поля).
type JTBDPrompt struct {
	consumer string
	text     string
	budget   int
}

func NewJTBDPrompt(consumer, text string, budget int) JTBDPrompt {
	return JTBDPrompt{consumer: consumer, text: text, budget: budget}
}

func (p JTBDPrompt) Consumer() string { return p.consumer }
func (p JTBDPrompt) Text() string     { return p.text }
func (p JTBDPrompt) Budget() int      { return p.budget }

// JTBDPromptSet — набор четырёх JTBDPrompt, один вход LLMClient.Simulate.
type JTBDPromptSet struct {
	prompts []JTBDPrompt
}

func NewJTBDPromptSet(prompts []JTBDPrompt) JTBDPromptSet {
	return JTBDPromptSet{prompts: prompts}
}

func (s JTBDPromptSet) Prompts() []JTBDPrompt { return s.prompts }

// LLMVerdict — сырой провайдер-агностичный вердикт от LLMClient.Simulate.
type LLMVerdict struct {
	Consumer  string
	RawStatus string
	RawScore  int
	RawGaps   []string
}

// ── Данные (I/O-выход и промежуточные) ──────────────────────────────────────

// Heading — заголовок H1–H6 в Markdown-документе.
type Heading struct {
	Level int
	Text  string
	Line  int
}

// MarkdownDoc — прочитанный Markdown-файл.
type MarkdownDoc struct {
	Path     string
	Lines    []string
	Headings []Heading
}

// RepoStructure — сырые факты ФС для L3/L6a.
type RepoStructure struct {
	Files     []string
	Docs      []MarkdownDoc
	MTimes    map[string]time.Time
	Manifests map[string]string
}

// LayerResult — результат слоя (сериализуется в JSON).
type LayerResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Score   *int   `json:"score,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// LayerOutcome — единый результат слоя без ИИ-интеграций.
type LayerOutcome struct {
	Result     LayerResult
	Violations []Violation
}

// Violation — конкретное нарушение.
type Violation struct {
	Code     string `json:"code"`
	Layer    string `json:"layer"`
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     *int   `json:"line,omitempty"`
	Message  string `json:"message"`
}

// Error — ошибка запуска (код возврата 2).
type Error struct {
	Code        string  `json:"code"`
	Integration *string `json:"integration,omitempty"`
	Message     string  `json:"message"`
}

// JTBDResult — результат по потребителю.
type JTBDResult struct {
	Status string   `json:"status"`
	Score  int      `json:"score"`
	Gaps   []string `json:"gaps"`
}

// ── Отчёт ────────────────────────────────────────────────────────────────────

// ReportTarget — target-секция отчёта.
type ReportTarget struct {
	Path   string  `json:"path"`
	Commit *string `json:"commit,omitempty"`
}

// Report — агрегат, сериализуется по report.schema.json.
type Report struct {
	SchemaVersion string                 `json:"schema_version"`
	Tool          string                 `json:"tool"`
	Command       string                 `json:"command"`
	Target        ReportTarget           `json:"target"`
	Layers        map[string]LayerResult `json:"layers,omitempty"`
	JTBD          map[string]JTBDResult  `json:"jtbd,omitempty"`
	Violations    []Violation            `json:"violations,omitempty"`
	Errors        []Error                `json:"errors,omitempty"`
}

// ReportParts — сборочный DTO для buildReport.
type ReportParts struct {
	Layers []LayerOutcome
	JTBD   []JTBDResult
}

// ── вспомогательные ──────────────────────────────────────────────────────────

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
