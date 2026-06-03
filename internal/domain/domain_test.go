package domain_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// writeGitHead создаёт минимальную структуру .git/ для тестов headCommit.
func writeGitHead(t *testing.T, root, content string) {
	t.Helper()
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile HEAD: %v", err)
	}
}

// ── NewAuditTarget / headCommit ───────────────────────────────────────────────

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

// filesAndManifestsYAML — обязательные секции required_files, manifests и
// link_extensions для фикстур (все три обязательны в кастомном конфиге).
const filesAndManifestsYAML = "required_files: [README.md]\nmanifests: [go.mod]\nlink_extensions: [md, go, sh]\n"

// TestNewConfig_jtbdFromConfig фиксирует: словари секций L4 берутся из YAML,
// роли не хардкодятся в Go.
func TestNewConfig_jtbdFromConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := "jtbd:\n  consumers:\n    - role: custom\n      sections:\n        - name: раздел\n          synonyms: [раздел, section]\n          critical: false\n" +
		filesAndManifestsYAML
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

// TestNewConfig_filesAndManifestsFromConfig фиксирует: required_files и manifests
// берутся из YAML (не хардкод в Go).
func TestNewConfig_filesAndManifestsFromConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := minimalJTBDYAML +
		"required_files: [README.md, LICENSE]\nmanifests: [go.mod, deno.json]\nlink_extensions: [md, go]\n"
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := domain.NewConfig(domain.Request{ConfigPath: cfgPath})
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	if got := cfg.RequiredFiles(); len(got) != 2 || got[1] != "LICENSE" {
		t.Errorf("RequiredFiles = %v, want [README.md LICENSE]", got)
	}
	if got := cfg.Manifests(); len(got) != 2 || got[1] != "deno.json" {
		t.Errorf("Manifests = %v, want [go.mod deno.json]", got)
	}
}

// TestNewConfig_requiredSectionsMissing фиксирует: кастомный конфиг без
// required_files или без manifests → config_invalid (не тихая деградация).
func TestNewConfig_requiredSectionsMissing(t *testing.T) {
	cases := map[string]string{
		"без required_files": minimalJTBDYAML + "manifests: [go.mod]\n",
		"без manifests":      minimalJTBDYAML + "required_files: [README.md]\n",
	}
	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			cfgPath := filepath.Join(dir, "config.yaml")
			if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := domain.NewConfig(domain.Request{ConfigPath: cfgPath})
			if !errors.Is(err, domain.ErrConfigInvalid) {
				t.Fatalf("expected ErrConfigInvalid, got %v", err)
			}
		})
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
		minimalJTBDYAML + filesAndManifestsYAML
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

// ── headCommit (через NewAuditTarget) ─────────────────────────────────────────

func TestHeadCommit_detached(t *testing.T) {
	dir := t.TempDir()
	const hash = "abc1234567890123456789012345678901234567"
	writeGitHead(t, dir, hash+"\n")

	target, err := domain.NewAuditTarget(domain.Request{Path: dir})
	if err != nil {
		t.Fatalf("NewAuditTarget: %v", err)
	}
	if got := target.Commit(); got != hash[:40] {
		t.Errorf("Commit() = %q, want %q", got, hash[:40])
	}
}

func TestHeadCommit_branchRef(t *testing.T) {
	dir := t.TempDir()
	const hash = "deadbeef1234567890123456789012345678abcd"
	writeGitHead(t, dir, "ref: refs/heads/main\n")

	// Создаём refs/heads/main с хэшем.
	refsDir := filepath.Join(dir, ".git", "refs", "heads")
	if err := os.MkdirAll(refsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(refsDir, "main"), []byte(hash+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	target, err := domain.NewAuditTarget(domain.Request{Path: dir})
	if err != nil {
		t.Fatalf("NewAuditTarget: %v", err)
	}
	if got := target.Commit(); got != hash[:40] {
		t.Errorf("Commit() = %q, want %q (должен разыменовать ref)", got, hash[:40])
	}
}

func TestHeadCommit_packedRef(t *testing.T) {
	dir := t.TempDir()
	const hash = "cafe0000000000000000000000000000000000ff"
	writeGitHead(t, dir, "ref: refs/heads/feature\n")

	// Нет loose ref — только packed-refs.
	packed := "# pack-refs with: peeled fully-peeled sorted\n" +
		hash + " refs/heads/feature\n"
	if err := os.WriteFile(filepath.Join(dir, ".git", "packed-refs"), []byte(packed), 0o644); err != nil {
		t.Fatalf("WriteFile packed-refs: %v", err)
	}

	target, err := domain.NewAuditTarget(domain.Request{Path: dir})
	if err != nil {
		t.Fatalf("NewAuditTarget: %v", err)
	}
	if got := target.Commit(); got != hash[:40] {
		t.Errorf("Commit() = %q, want %q (из packed-refs)", got, hash[:40])
	}
}

func TestHeadCommit_noGit(t *testing.T) {
	dir := t.TempDir()
	// Нет .git/ вообще → Commit() пустой, не паника.
	target, err := domain.NewAuditTarget(domain.Request{Path: dir})
	if err != nil {
		t.Fatalf("NewAuditTarget: %v", err)
	}
	if got := target.Commit(); got != "" {
		t.Errorf("Commit() = %q, want empty (нет .git)", got)
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

// ── LinkExtensions ────────────────────────────────────────────────────────────

func TestNewConfig_linkExtensionsFromDefault(t *testing.T) {
	cfg, err := domain.NewConfig(domain.Request{})
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	exts := cfg.LinkExtensions()
	if len(exts) == 0 {
		t.Fatal("LinkExtensions() не должен быть пустым из дефолтного конфига")
	}
	want := map[string]bool{"md": false, "go": false, "sh": false, "yml": false}
	for _, e := range exts {
		want[e] = true
	}
	for ext, found := range want {
		if !found {
			t.Errorf("LinkExtensions(): ожидали расширение %q в дефолтном конфиге", ext)
		}
	}
}

func TestNewConfig_linkExtensionsMissing(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := minimalJTBDYAML + "required_files: [README.md]\nmanifests: [go.mod]\n"
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := domain.NewConfig(domain.Request{ConfigPath: cfgPath})
	if !errors.Is(err, domain.ErrConfigInvalid) {
		t.Fatalf("кастомный конфиг без link_extensions: expected ErrConfigInvalid, got %v", err)
	}
}
