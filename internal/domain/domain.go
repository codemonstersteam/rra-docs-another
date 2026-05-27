// Package domain содержит типы и конструкторы доменной модели rra-docs-another.
// Валидируемые структуры имеют неэкспортируемые поля и создаются конструктором.
// Остальные — плоские DTO (публичные поля).
package domain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

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

// Root возвращает абсолютный путь к корню репозитория.
func (t AuditTarget) Root() string { return t.root }

// Commit возвращает HEAD, если это git-репо, иначе "".
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
	// Проверяем права чтения через ReadDir.
	if _, err := os.ReadDir(abs); err != nil {
		return AuditTarget{}, fmt.Errorf("%w: %s", ErrReadError, abs)
	}
	commit := headCommit(abs)
	return AuditTarget{root: abs, commit: commit}, nil
}

// headCommit пытается прочитать HEAD-коммит из .git/HEAD.
func headCommit(root string) string {
	data, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	if err != nil {
		return ""
	}
	head := string(data)
	// Если HEAD указывает на ref, возвращаем строку как есть (укороченная форма).
	// Для целей S1 достаточно непустой строки.
	if len(head) > 0 {
		return head[:min(len(head), 40)]
	}
	return ""
}

// Config — валидированный проектный конфиг (неэкспортируемые поля).
type Config struct {
	driftThresholdDays int
	readabilityMin     int
}

// DriftThresholdDays — порог устаревания документации в днях.
func (c Config) DriftThresholdDays() int { return c.driftThresholdDays }

// ReadabilityMin — минимальный допустимый балл читаемости (Flesch Reading Ease, 0–100).
// Значения ниже порога — предупреждение (warning), не блокер.
func (c Config) ReadabilityMin() int { return c.readabilityMin }

// NewConfig валидирует и создаёт Config.
// Если ConfigPath пуст — берутся встроенные дефолты.
// Failure: ErrConfigInvalid.
func NewConfig(req Request) (Config, error) {
	if req.ConfigPath == "" {
		return Config{driftThresholdDays: 90, readabilityMin: 50}, nil
	}
	cfg, err := parseConfigFile(req.ConfigPath)
	if err != nil {
		return Config{}, fmt.Errorf("%w: %s", ErrConfigInvalid, err)
	}
	return cfg, nil
}

// parseConfigFile читает конфиг-файл. Сейчас поддерживается только дефолт
// (расширение до YAML/TOML — в будущих слайсах).
func parseConfigFile(path string) (Config, error) {
	if _, err := os.Stat(path); err != nil {
		return Config{}, fmt.Errorf("файл не найден: %s", path)
	}
	// TODO(S3+): парсинг YAML/TOML схемы конфига.
	return Config{driftThresholdDays: 90, readabilityMin: 50}, nil
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
