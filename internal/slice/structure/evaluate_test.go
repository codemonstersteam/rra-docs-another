package structure_test

import (
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/structure"
)

func TestEvaluate_pass(t *testing.T) {
	cfg := structure.ExportMakeConfig(90)
	s := domain.RepoStructure{
		Files: []string{"README.md", "main.go"},
		Docs:  []domain.MarkdownDoc{{Path: "README.md", Lines: []string{"# Hello"}}},
	}
	outcome := structure.Evaluate(s, cfg)
	if outcome.Result.Status != "pass" {
		t.Errorf("status = %q, want pass", outcome.Result.Status)
	}
	if len(outcome.Violations) != 0 {
		t.Errorf("violations = %v, want none", outcome.Violations)
	}
}

func TestEvaluate_failMissingRequired(t *testing.T) {
	cfg := structure.ExportMakeConfig(90)
	s := domain.RepoStructure{Files: []string{"main.go"}}
	outcome := structure.Evaluate(s, cfg)
	if outcome.Result.Status != "fail" {
		t.Errorf("status = %q, want fail", outcome.Result.Status)
	}
	if len(outcome.Violations) == 0 {
		t.Error("expected violations, got none")
	}
}

func TestEvaluate_warnBrokenLink(t *testing.T) {
	cfg := structure.ExportMakeConfig(90)
	s := domain.RepoStructure{
		Files: []string{"README.md"},
		Docs: []domain.MarkdownDoc{
			{Path: "README.md", Lines: []string{"[missing](docs/missing.md)"}},
		},
	}
	outcome := structure.Evaluate(s, cfg)
	// broken_link — blocker → fail
	if outcome.Result.Status != "fail" {
		t.Errorf("status = %q, want fail (broken_link is blocker)", outcome.Result.Status)
	}
}
