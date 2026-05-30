package fitness

import (
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// ── buildJTBDPromptSet ───────────────────────────────────────────────────────

func TestBuildJTBDPromptSet_happy(t *testing.T) {
	cfg, err := domain.NewConfig(domain.Request{})
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	docs := []domain.MarkdownDoc{
		{Path: "README.md", Lines: []string{"# Hello", "World"}},
	}
	set := buildJTBDPromptSet(docs, cfg)
	if len(set.Prompts()) != 4 {
		t.Fatalf("expected 4 prompts, got %d", len(set.Prompts()))
	}
	for _, p := range set.Prompts() {
		if p.Consumer() == "" {
			t.Error("consumer must not be empty")
		}
		if p.Text() == "" {
			t.Error("text must not be empty")
		}
		if p.Budget() <= 0 {
			t.Error("budget must be positive")
		}
	}
}

func TestBuildJTBDPromptSet_emptyDocs(t *testing.T) {
	cfg, err := domain.NewConfig(domain.Request{})
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	set := buildJTBDPromptSet(nil, cfg)
	if len(set.Prompts()) != 4 {
		t.Fatalf("expected 4 prompts even for empty docs, got %d", len(set.Prompts()))
	}
}

// ── scoreFitness ─────────────────────────────────────────────────────────────

func TestScoreFitness_happy(t *testing.T) {
	verdicts := []domain.LLMVerdict{
		{Consumer: "maintainer", RawStatus: "PASS", RawScore: 92, RawGaps: nil},
		{Consumer: "consumer", RawStatus: "PARTIAL", RawScore: 55, RawGaps: []string{"missing"}},
	}
	results := scoreFitness(verdicts)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Status != "PASS" || results[0].Score != 92 {
		t.Errorf("result[0] = %+v", results[0])
	}
	if results[1].Status != "PARTIAL" || results[1].Score != 55 {
		t.Errorf("result[1] = %+v", results[1])
	}
}

func TestScoreFitness_invalidStatusBecomesPartial(t *testing.T) {
	verdicts := []domain.LLMVerdict{
		{Consumer: "agent", RawStatus: "UNKNOWN_STATUS", RawScore: 30},
	}
	results := scoreFitness(verdicts)
	if results[0].Status != "PARTIAL" {
		t.Errorf("invalid status should become PARTIAL, got %q", results[0].Status)
	}
}
