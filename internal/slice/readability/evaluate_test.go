package readability_test

import (
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/readability"
)

func TestEvaluate_pass(t *testing.T) {
	cfg := readability.ExportMakeConfig(50)
	// Высокочитаемый текст (короткие предложения).
	docs := []domain.MarkdownDoc{
		{Path: "README.md", Lines: []string{"This is easy. Short words. Good text."}},
	}
	outcome := readability.Evaluate(docs, cfg)
	if outcome.Result.Status != "pass" {
		t.Errorf("status = %q, want pass", outcome.Result.Status)
	}
}

func TestEvaluate_warnLowReadability(t *testing.T) {
	cfg := readability.ExportMakeConfig(90) // очень высокий порог
	docs := []domain.MarkdownDoc{
		{Path: "arch.md", Lines: []string{
			"The extraordinarily convoluted architectural specification establishes prerequisites.",
		}},
	}
	outcome := readability.Evaluate(docs, cfg)
	if outcome.Result.Status != "warn" {
		t.Errorf("status = %q, want warn", outcome.Result.Status)
	}
}

func TestEvaluate_emptyDocs(t *testing.T) {
	cfg := readability.ExportMakeConfig(50)
	outcome := readability.Evaluate(nil, cfg)
	if outcome.Result.Status != "pass" {
		t.Errorf("empty docs: status = %q, want pass", outcome.Result.Status)
	}
}
