package domain_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// ── NewAuditTarget ───────────────────────────────────────────────────────────

func TestNewAuditTarget_happy(t *testing.T) {
	dir := t.TempDir()
	req := domain.Request{Path: dir}
	got, err := domain.NewAuditTarget(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Root() == "" {
		t.Error("Root must not be empty")
	}
}

func TestNewAuditTarget_pathNotFound(t *testing.T) {
	req := domain.Request{Path: "/no-such-dir-zzzzzz"}
	_, err := domain.NewAuditTarget(req)
	if !errors.Is(err, domain.ErrPathNotFound) {
		t.Fatalf("expected ErrPathNotFound, got %v", err)
	}
}

func TestNewAuditTarget_noPerms(t *testing.T) {
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Skip("cannot create unreadable dir:", err)
	}
	defer os.Chmod(locked, 0o700) //nolint:errcheck

	if os.Getuid() == 0 {
		t.Skip("root ignores permission bits")
	}
	req := domain.Request{Path: locked}
	_, err := domain.NewAuditTarget(req)
	if !errors.Is(err, domain.ErrReadError) {
		t.Fatalf("expected ErrReadError, got %v", err)
	}
}

// ── NewConfig ────────────────────────────────────────────────────────────────

func TestNewConfig_happy(t *testing.T) {
	req := domain.Request{}
	cfg, err := domain.NewConfig(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DriftThresholdDays() <= 0 {
		t.Error("DriftThresholdDays must be positive")
	}
}

func TestNewConfig_badConfig(t *testing.T) {
	req := domain.Request{ConfigPath: "/no-such-config-file-zzzz.yaml"}
	_, err := domain.NewConfig(req)
	if !errors.Is(err, domain.ErrConfigInvalid) {
		t.Fatalf("expected ErrConfigInvalid, got %v", err)
	}
}

// minimalJTBDYAML — минимальная валидная секция jtbd для фикстур,
// проверяющих другие части конфига (jtbd обязательна в кастомном конфиге).
const minimalJTBDYAML = "jtbd:\n  consumers:\n    - role: maintainer\n      sections:\n        - name: архитектура\n          synonyms: [архитектура]\n          critical: true\n"

// TestNewConfig_jtbdFromConfig фиксирует: словари секций L4 берутся из YAML,
// роли не хардкодятся в Go.
func TestNewConfig_jtbdFromConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := "jtbd:\n  consumers:\n    - role: custom\n      sections:\n        - name: раздел\n          synonyms: [раздел, section]\n          critical: false\n"
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := domain.NewConfig(domain.Request{ConfigPath: cfgPath})
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	consumers := cfg.JTBDSpec().Consumers()
	if len(consumers) != 1 || consumers[0].Role() != "custom" {
		t.Fatalf("ожидали 1 роль custom, получили %+v", consumers)
	}
	secs := consumers[0].Sections()
	if len(secs) != 1 || secs[0].Name() != "раздел" || secs[0].Critical() {
		t.Errorf("секция распарсилась неверно: %+v", secs)
	}
}

// TestNewConfig_jtbdMissing фиксирует: кастомный конфиг без секции jtbd →
// config_invalid (решение оператора, не тихий PASS).
func TestNewConfig_jtbdMissing(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("thresholds:\n  drift_days: 30\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := domain.NewConfig(domain.Request{ConfigPath: cfgPath})
	if !errors.Is(err, domain.ErrConfigInvalid) {
		t.Fatalf("expected ErrConfigInvalid, got %v", err)
	}
}

// ── NewLLMConfig ─────────────────────────────────────────────────────────────

func TestNewLLMConfig_happy(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	req := domain.Request{LLMProvider: "openai", LLMBaseURL: "http://localhost:8080"}
	cfg, err := domain.NewLLMConfig(req, domain.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Provider() != "openai" {
		t.Errorf("provider = %q, want openai", cfg.Provider())
	}
	if cfg.BaseURL() != "http://localhost:8080" {
		t.Errorf("base url = %q, флаг должен победить", cfg.BaseURL())
	}
	if cfg.APIKey() != "test-key" {
		t.Errorf("api key not set")
	}
}

// TestNewLLMConfig_anthropicDefaultBaseURLV1 фиксирует: дефолт anthropic — с /v1
// (а не голый домен). Эту нестыковку устраняли отдельно.
func TestNewLLMConfig_anthropicDefaultBaseURLV1(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	cfg, err := domain.NewLLMConfig(domain.Request{LLMProvider: "anthropic"}, domain.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL() != "https://api.anthropic.com/v1" {
		t.Errorf("base url = %q, want https://api.anthropic.com/v1", cfg.BaseURL())
	}
}

// TestNewLLMConfig_baseURLFromConfigLayer фиксирует приоритет YAML-конфига над
// дефолтом (флаг > файл > дефолт): без флага base_url берётся из конфига.
func TestNewLLMConfig_baseURLFromConfigLayer(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := "llm:\n  provider: openai\n  base_url: http://cfg-host:9999/v1\n  model: cfg-model\n" +
		minimalJTBDYAML
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := domain.NewConfig(domain.Request{ConfigPath: cfgPath})
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	llmCfg, err := domain.NewLLMConfig(domain.Request{}, cfg) // без флагов
	if err != nil {
		t.Fatalf("NewLLMConfig: %v", err)
	}
	if llmCfg.Provider() != "openai" {
		t.Errorf("provider = %q, want openai (из конфига)", llmCfg.Provider())
	}
	if llmCfg.BaseURL() != "http://cfg-host:9999/v1" {
		t.Errorf("base url = %q, want из конфига", llmCfg.BaseURL())
	}
	if llmCfg.Model() != "cfg-model" {
		t.Errorf("model = %q, want cfg-model (из конфига)", llmCfg.Model())
	}
}

func TestNewLLMConfig_noKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	os.Unsetenv("ANTHROPIC_API_KEY")
	req := domain.Request{LLMProvider: "anthropic"}
	_, err := domain.NewLLMConfig(req, domain.Config{})
	if !errors.Is(err, domain.ErrLLMUnavailable) {
		t.Fatalf("expected ErrLLMUnavailable, got %v", err)
	}
}

func TestNewLLMConfig_openaiNoBaseURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	req := domain.Request{LLMProvider: "openai", LLMBaseURL: ""}
	_, err := domain.NewLLMConfig(req, domain.Config{})
	if !errors.Is(err, domain.ErrLLMUnavailable) {
		t.Fatalf("expected ErrLLMUnavailable, got %v", err)
	}
}
