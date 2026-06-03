package fitness

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// stubAsker — тестовый заменитель LLM.
type stubAsker struct {
	verdicts []domain.LLMVerdict
	err      error
}

func (s stubAsker) Ask(_ domain.JTBDPromptSet) ([]domain.LLMVerdict, error) {
	return s.verdicts, s.err
}

func defaultCfg(t *testing.T) domain.Config {
	t.Helper()
	cfg, err := domain.NewConfig(domain.Request{})
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	return cfg
}

// ── Evaluate happy path ───────────────────────────────────────────────────────

func TestEvaluate_happy(t *testing.T) {
	cfg := defaultCfg(t)
	docs := []domain.MarkdownDoc{
		{Path: "README.md", Lines: []string{"# Hello"}},
	}
	verdicts := []domain.LLMVerdict{
		{Consumer: "maintainer", RawStatus: "PASS", RawScore: 90},
		{Consumer: "consumer", RawStatus: "PARTIAL", RawScore: 60, RawGaps: []string{"missing"}},
		{Consumer: "manager", RawStatus: "PASS", RawScore: 85},
		{Consumer: "agent", RawStatus: "PASS", RawScore: 80},
	}
	result, err := Evaluate(docs, cfg, stubAsker{verdicts: verdicts})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(result) != 4 {
		t.Fatalf("len(result) = %d, want 4", len(result))
	}
	if r := result["maintainer"]; r.Status != "PASS" || r.Score != 90 {
		t.Errorf("maintainer = %+v", r)
	}
}

// ── Evaluate filter branch ────────────────────────────────────────────────────

func TestEvaluate_filterByDocsList(t *testing.T) {
	cfgYAML := `
docs: [README.md]
required_files: [README.md]
manifests: [go.mod]
link_extensions: [md, go, sh]
thresholds:
  drift_days: 90
  readability_min: 50
llm:
  provider: anthropic
  api_key_env: ANTHROPIC_API_KEY
  token_budget: 300000
  max_judge_calls: 20
prompts:
  maintainer: "rate docs"
  consumer: "rate docs"
  manager: "rate docs"
  agent: "rate docs"
jtbd:
  consumers:
    - role: maintainer
      sections:
        - name: Installation
          synonyms: [install]
          critical: true
    - role: consumer
      sections:
        - name: Usage
          synonyms: [usage]
          critical: true
    - role: manager
      sections:
        - name: Overview
          synonyms: [overview]
          critical: false
    - role: agent
      sections:
        - name: API
          synonyms: [api]
          critical: false
`
	f, err := os.CreateTemp(t.TempDir(), "cfg-*.yaml")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err = f.WriteString(cfgYAML); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	f.Close()

	cfg, err := domain.NewConfig(domain.Request{ConfigPath: f.Name()})
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}

	docs := []domain.MarkdownDoc{
		{Path: "README.md", Lines: []string{"# Hello"}},
		{Path: "EXTRA.md", Lines: []string{"# Extra content not in filter"}},
	}

	verdicts := []domain.LLMVerdict{
		{Consumer: "maintainer", RawStatus: "PASS", RawScore: 80},
		{Consumer: "consumer", RawStatus: "PASS", RawScore: 80},
		{Consumer: "manager", RawStatus: "PASS", RawScore: 80},
		{Consumer: "agent", RawStatus: "PASS", RawScore: 80},
	}

	var capturedText string
	capture := captureAsker{
		fn:       func(set domain.JTBDPromptSet) { capturedText = promptsText(set) },
		verdicts: verdicts,
	}

	if _, err = Evaluate(docs, cfg, capture); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if strings.Contains(capturedText, "EXTRA.md") {
		t.Error("промпты содержат EXTRA.md, хотя он вне фильтра cfg.Docs()")
	}
	if !strings.Contains(capturedText, "README.md") {
		t.Error("промпты не содержат README.md")
	}
}

// ── Evaluate error branch ─────────────────────────────────────────────────────

func TestEvaluate_llmError(t *testing.T) {
	cfg := defaultCfg(t)
	sentinelErr := errors.New("llm down")
	_, err := Evaluate(nil, cfg, stubAsker{err: sentinelErr})
	if !errors.Is(err, sentinelErr) {
		t.Errorf("err = %v, want %v", err, sentinelErr)
	}
}

// ── вспомогательные типы ──────────────────────────────────────────────────────

type captureAsker struct {
	fn       func(domain.JTBDPromptSet)
	verdicts []domain.LLMVerdict
}

func (c captureAsker) Ask(set domain.JTBDPromptSet) ([]domain.LLMVerdict, error) {
	c.fn(set)
	return c.verdicts, nil
}

func promptsText(set domain.JTBDPromptSet) string {
	var sb strings.Builder
	for _, p := range set.Prompts() {
		sb.WriteString(p.Text())
	}
	return sb.String()
}
