package drift_test

import (
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
	iodep "github.com/codemonstersteam/rra-docs-another/internal/io"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/drift"
)

func defaultDriftCfg(t *testing.T) domain.Config {
	t.Helper()
	cfg, err := domain.NewConfig(domain.Request{})
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	return cfg
}

func TestEvaluate_pass(t *testing.T) {
	cfg := defaultDriftCfg(t)
	s := domain.RepoStructure{
		Files: []string{"README.md"},
		Docs: []domain.MarkdownDoc{
			{Path: "README.md", Lines: []string{"# Hello world"}},
		},
	}
	outcome, err := drift.Evaluate(s, cfg, iodep.NoopJudge{})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if outcome.Result.Status != "pass" {
		t.Errorf("status = %q, want pass", outcome.Result.Status)
	}
	if len(outcome.Violations) != 0 {
		t.Errorf("violations = %v, want none", outcome.Violations)
	}
}

func TestEvaluate_failBrokenLink(t *testing.T) {
	cfg := defaultDriftCfg(t)
	s := domain.RepoStructure{
		Files: []string{"README.md"},
		Docs: []domain.MarkdownDoc{
			{
				Path:  "README.md",
				Lines: []string{"See `internal/missing/file.go` for details."},
			},
		},
	}
	outcome, err := drift.Evaluate(s, cfg, iodep.NoopJudge{})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if outcome.Result.Status != "fail" {
		t.Errorf("status = %q, want fail (broken path claim)", outcome.Result.Status)
	}
	if len(outcome.Violations) == 0 {
		t.Error("expected violations, got none")
	}
}

func TestEvaluate_judgeError(t *testing.T) {
	cfg := defaultDriftCfg(t)
	s := domain.RepoStructure{Files: []string{"README.md"}}
	errJudge := errorJudge{}
	_, err := drift.Evaluate(s, cfg, errJudge)
	if err == nil {
		t.Error("expected error from judge, got nil")
	}
}

// errorJudge — судья, всегда возвращающий ошибку.
type errorJudge struct{}

func (errorJudge) Judge(_ domain.ClaimPromptSet) ([]domain.Verdict, error) {
	return nil, domain.ErrLLMUnavailable
}
