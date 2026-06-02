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
		Provider      string `yaml:"provider"`
		Model         string `yaml:"model"`
		APIKeyEnv     string `yaml:"api_key_env"`
		BaseURL       string `yaml:"base_url"`
		CallDelayMs   int    `yaml:"call_delay_ms"`
		TokenBudget   int    `yaml:"token_budget"`
		MaxRetries    int    `yaml:"max_retries"`
		MaxJudgeCalls int    `yaml:"max_judge_calls"`
	} `yaml:"llm"`
	Docs          []string          `yaml:"docs"`
	RequiredFiles []string          `yaml:"required_files"`
	Manifests     []string          `yaml:"manifests"`
	Prompts       map[string]string `yaml:"prompts"`
	Thresholds    struct {
		DriftDays      int `yaml:"drift_days"`
		ReadabilityMin int `yaml:"readability_min"`
	} `yaml:"thresholds"`
	JTBD struct {
		Consumers []struct {
			Role     string `yaml:"role"`
			Sections []struct {
				Name     string   `yaml:"name"`
				Synonyms []string `yaml:"synonyms"`
				Critical bool     `yaml:"critical"`
			} `yaml:"sections"`
		} `yaml:"consumers"`
	} `yaml:"jtbd"`
}

// Config — валидированный проектный конфиг (неэкспортируемые поля).
type Config struct {
	driftThresholdDays int
	readabilityMin     int
	llmPrompts         map[string]string
	docs               []string
	requiredFiles      []string
	manifests          []string
	llmCallDelayMs     int
	llmTokenBudget     int
	llmMaxRetries      int
	llmProvider        string
	llmBaseURL         string
	llmModel           string
	maxJudgeCalls      int
	jtbdSpec           JTBDSpec
}

// LLMProvider/LLMBaseURL/LLMModel — слой YAML-конфига для LLM-подключения
// («файл» в приоритете флаг > файл > вшитый дефолт, ADR 0003). Пусто = слой
// не задан, резолвится во флаге или дефолте (см. NewLLMConfig).
func (c Config) LLMProvider() string { return c.llmProvider }
func (c Config) LLMBaseURL() string  { return c.llmBaseURL }
func (c Config) LLMModel() string    { return c.llmModel }

func (c Config) DriftThresholdDays() int { return c.driftThresholdDays }
func (c Config) ReadabilityMin() int     { return c.readabilityMin }

// Docs возвращает список doc-файлов для проверки (относительные пути от корня репо).
func (c Config) Docs() []string { return c.docs }

// RequiredFiles возвращает обязательные файлы в корне репо (L3, слайс structure).
func (c Config) RequiredFiles() []string { return c.requiredFiles }

// Manifests возвращает известные файлы-манифесты для разбора зависимостей
// (claim-kind dependency, L6, слайс drift).
func (c Config) Manifests() []string { return c.manifests }

// LLMCallDelayMs возвращает задержку между последовательными LLM-вызовами (мс).
// 0 = без задержки (дефолт для тестов). Для реального API рекомендуется 10000.
func (c Config) LLMCallDelayMs() int { return c.llmCallDelayMs }

// LLMTokenBudget возвращает защитный лимит токенов на один вызов (usage.total_tokens).
// Предохранитель от аномалий — выше реального максимума целевых репо, не ниже
// (skill http-io → «Бюджет payload»). Дефолт 300000.
func (c Config) LLMTokenBudget() int { return c.llmTokenBudget }

// LLMMaxRetries возвращает число повторов на transient-отказ (429) с бэкоффом
// по Retry-After. 0 = без повтора (дефолт; см. skill http-io → «Пацинг»).
func (c Config) LLMMaxRetries() int { return c.llmMaxRetries }

// MaxJudgeCalls возвращает максимальное число вызовов Judge на один запуск (L6c).
// Ограничивает нагрузку на LLM при наличии флага --semantic.
func (c Config) MaxJudgeCalls() int { return c.maxJudgeCalls }

// LLMPrompt возвращает промпт для роли (maintainer|consumer|manager|agent).
func (c Config) LLMPrompt(role string) string {
	if c.llmPrompts == nil {
		return ""
	}
	return c.llmPrompts[role]
}

// JTBDSpec возвращает словарь обязательных секций по JTBD-ролям (слой L4).
// Узкий срез Config: голова слайса jtbd передаёт его в чистую логику,
// не таская весь Config.
func (c Config) JTBDSpec() JTBDSpec { return c.jtbdSpec }

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
	tb := raw.LLM.TokenBudget
	if tb == 0 {
		tb = 300_000
	}
	mjc := raw.LLM.MaxJudgeCalls
	if mjc == 0 {
		mjc = 20
	}
	jtbdSpec, err := buildJTBDSpec(raw)
	if err != nil {
		return Config{}, err
	}
	if len(raw.RequiredFiles) == 0 {
		return Config{}, fmt.Errorf("%w: секция required_files пуста или отсутствует", ErrConfigInvalid)
	}
	if len(raw.Manifests) == 0 {
		return Config{}, fmt.Errorf("%w: секция manifests пуста или отсутствует", ErrConfigInvalid)
	}
	return Config{
		driftThresholdDays: dt,
		readabilityMin:     rm,
		llmPrompts:         raw.Prompts,
		docs:               raw.Docs,
		requiredFiles:      raw.RequiredFiles,
		manifests:          raw.Manifests,
		llmCallDelayMs:     raw.LLM.CallDelayMs,
		llmTokenBudget:     tb,
		llmMaxRetries:      raw.LLM.MaxRetries,
		llmProvider:        raw.LLM.Provider,
		llmBaseURL:         raw.LLM.BaseURL,
		llmModel:           raw.LLM.Model,
		maxJudgeCalls:      mjc,
		jtbdSpec:           jtbdSpec,
	}, nil
}

// ── JTBDSpec — словари секций L4 ─────────────────────────────────────────────

// JTBDSpec — словарь обязательных секций по JTBD-ролям (неэкспортируемые поля).
// Создаётся из YAML внутри NewConfig (buildJTBDSpec), доступен через Config.JTBDSpec().
type JTBDSpec struct {
	consumers []JTBDConsumer
}

// JTBDConsumer — набор обязательных секций для одной JTBD-роли.
type JTBDConsumer struct {
	role     string
	sections []JTBDSection
}

// JTBDSection — обязательная секция: хотя бы один synonym должен входить
// (как подстрока) в нормализованный заголовок документа.
type JTBDSection struct {
	name     string
	synonyms []string
	critical bool
}

func (s JTBDSpec) Consumers() []JTBDConsumer   { return s.consumers }
func (c JTBDConsumer) Role() string            { return c.role }
func (c JTBDConsumer) Sections() []JTBDSection { return c.sections }
func (s JTBDSection) Name() string             { return s.name }
func (s JTBDSection) Synonyms() []string       { return s.synonyms }
func (s JTBDSection) Critical() bool           { return s.critical }

// buildJTBDSpec валидирует raw-секцию jtbd и собирает JTBDSpec.
// Антецедент: ≥1 consumer; у каждого непустой role и ≥1 section; у каждой
// section непустой name и ≥1 synonym. Роли не фиксированы — берутся из конфига.
// Failure: ErrConfigInvalid (в т.ч. отсутствие секции jtbd в кастомном конфиге).
func buildJTBDSpec(raw configYAML) (JTBDSpec, error) {
	if len(raw.JTBD.Consumers) == 0 {
		return JTBDSpec{}, fmt.Errorf("%w: секция jtbd пуста или отсутствует", ErrConfigInvalid)
	}
	consumers := make([]JTBDConsumer, 0, len(raw.JTBD.Consumers))
	for _, rc := range raw.JTBD.Consumers {
		if rc.Role == "" {
			return JTBDSpec{}, fmt.Errorf("%w: jtbd-роль с пустым role", ErrConfigInvalid)
		}
		if len(rc.Sections) == 0 {
			return JTBDSpec{}, fmt.Errorf("%w: роль %q без секций", ErrConfigInvalid, rc.Role)
		}
		sections := make([]JTBDSection, 0, len(rc.Sections))
		for _, rs := range rc.Sections {
			if rs.Name == "" {
				return JTBDSpec{}, fmt.Errorf("%w: роль %q: секция без name", ErrConfigInvalid, rc.Role)
			}
			if len(rs.Synonyms) == 0 {
				return JTBDSpec{}, fmt.Errorf("%w: роль %q, секция %q без synonyms", ErrConfigInvalid, rc.Role, rs.Name)
			}
			sections = append(sections, JTBDSection{
				name:     rs.Name,
				synonyms: rs.Synonyms,
				critical: rs.Critical,
			})
		}
		consumers = append(consumers, JTBDConsumer{role: rc.Role, sections: sections})
	}
	return JTBDSpec{consumers: consumers}, nil
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

// NewLLMConfig валидирует LLM-подключение и создаёт LLMConfig — единственное
// место резолвинга baseURL/model/provider (приоритет флаг > YAML-конфиг > вшитый
// дефолт, ADR 0003). Клиент берёт готовые значения отсюда, ничего не хардкодит.
// Антецедент: provider ∈ {anthropic,openai}; baseURL непустой (для anthropic —
// дефолт https://api.anthropic.com/v1); ключ в env (ANTHROPIC_API_KEY | OPENAI_API_KEY).
// Failure: ErrLLMUnavailable.
func NewLLMConfig(req Request, cfg Config) (LLMConfig, error) {
	provider := firstNonEmpty(req.LLMProvider, cfg.LLMProvider(), "anthropic")
	if provider != "anthropic" && provider != "openai" {
		return LLMConfig{}, fmt.Errorf("%w: провайдер %q неизвестен", ErrLLMUnavailable, provider)
	}

	baseURL := firstNonEmpty(req.LLMBaseURL, cfg.LLMBaseURL())
	if baseURL == "" {
		if provider == "anthropic" {
			baseURL = "https://api.anthropic.com/v1"
		} else {
			return LLMConfig{}, fmt.Errorf("%w: openai требует base_url (--llm-base-url или llm.base_url)", ErrLLMUnavailable)
		}
	}

	envVar := "ANTHROPIC_API_KEY"
	if provider == "openai" {
		envVar = "OPENAI_API_KEY"
	}
	key := os.Getenv(envVar)
	if key == "" {
		return LLMConfig{}, fmt.Errorf("%w: переменная %s не задана", ErrLLMUnavailable, envVar)
	}

	model := firstNonEmpty(req.LLMModel, cfg.LLMModel())
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

// firstNonEmpty возвращает первую непустую строку (резолвинг слоёв конфига).
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
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

// JTBDPromptSet — набор четырёх JTBDPrompt, один вход LLMClient.Ask.
type JTBDPromptSet struct {
	prompts []JTBDPrompt
}

func NewJTBDPromptSet(prompts []JTBDPrompt) JTBDPromptSet {
	return JTBDPromptSet{prompts: prompts}
}

func (s JTBDPromptSet) Prompts() []JTBDPrompt { return s.prompts }

// LLMVerdict — сырой провайдер-агностичный вердикт от LLMClient.Ask.
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

// ── L6c — семантический судья ────────────────────────────────────────────────

// ClaimPrompt — одна пара (сниппет доки + сниппет кода) для семантического судьи.
type ClaimPrompt struct {
	DocSnippet  string
	CodeSnippet string
}

// ClaimPromptSet — набор пар для Judge.Judge (S6/S8).
type ClaimPromptSet struct {
	prompts []ClaimPrompt
}

// NewClaimPromptSet создаёт ClaimPromptSet из среза промптов.
func NewClaimPromptSet(prompts []ClaimPrompt) ClaimPromptSet {
	return ClaimPromptSet{prompts: prompts}
}

func (s ClaimPromptSet) Prompts() []ClaimPrompt { return s.prompts }
func (s ClaimPromptSet) IsEmpty() bool          { return len(s.prompts) == 0 }

// Verdict — вердикт судьи по одной паре (S6/S8).
type Verdict struct {
	OK    bool
	Quote string
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
