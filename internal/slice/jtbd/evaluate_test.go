package jtbd_test

import (
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/jtbd"
)

func TestEvaluate_allRolesPresent(t *testing.T) {
	cfg := jtbd.ExportMakeConfig()
	// Документ с заголовками, покрывающими все роли дефолтного конфига.
	docs := []domain.MarkdownDoc{
		{
			Path: "README.md",
			Lines: []string{
				"# Quick Start",
				"## Installation",
				"## Usage",
				"## API Reference",
				"## Architecture",
				"## Contributing",
				"## Changelog",
				"## Configuration",
				"## Deployment",
				"## Troubleshooting",
			},
			Headings: []domain.Heading{
				{Level: 1, Text: "Quick Start", Line: 1},
				{Level: 2, Text: "Installation", Line: 2},
				{Level: 2, Text: "Usage", Line: 3},
				{Level: 2, Text: "API Reference", Line: 4},
				{Level: 2, Text: "Architecture", Line: 5},
				{Level: 2, Text: "Contributing", Line: 6},
				{Level: 2, Text: "Changelog", Line: 7},
				{Level: 2, Text: "Configuration", Line: 8},
				{Level: 2, Text: "Deployment", Line: 9},
				{Level: 2, Text: "Troubleshooting", Line: 10},
			},
		},
	}
	result := jtbd.Evaluate(docs, cfg)
	if len(result) == 0 {
		t.Fatal("expected non-empty result map")
	}
	for role, r := range result {
		if r.Score < 0 || r.Score > 100 {
			t.Errorf("role %q: score %d out of range", role, r.Score)
		}
		if r.Status == "" {
			t.Errorf("role %q: empty status", role)
		}
		if r.Gaps == nil {
			t.Errorf("role %q: gaps must not be nil", role)
		}
	}
}

func TestEvaluate_emptyDocs(t *testing.T) {
	cfg := jtbd.ExportMakeConfig()
	result := jtbd.Evaluate(nil, cfg)
	for role, r := range result {
		if r.Status != "FAIL" {
			t.Errorf("role %q: empty docs should give FAIL, got %q", role, r.Status)
		}
		if r.Score != 0 {
			t.Errorf("role %q: score = %d, want 0", role, r.Score)
		}
	}
}
