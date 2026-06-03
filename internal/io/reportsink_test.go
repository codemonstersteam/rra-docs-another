package io

import (
	"strings"
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

func TestRenderMarkdown_jtbdSection(t *testing.T) {
	score := 85
	report := domain.Report{
		SchemaVersion: "1.0",
		Tool:          "rra-docs-another",
		Command:       "assess",
		Target:        domain.ReportTarget{Path: "/tmp/repo"},
		JTBD: map[string]domain.JTBDResult{
			"maintainer": {Status: "PASS", Score: 90, Gaps: []string{}},
			"consumer":   {Status: "PARTIAL", Score: 60, Gaps: []string{"quickstart"}},
			"manager":    {Status: "PASS", Score: score, Gaps: []string{}},
			"agent":      {Status: "FAIL", Score: 20, Gaps: []string{"AGENTS.md", "architecture"}},
		},
	}

	md := renderMarkdown(report)

	if !strings.Contains(md, "## JTBD") {
		t.Error("markdown должен содержать секцию ## JTBD")
	}
	for _, role := range []string{"maintainer", "consumer", "manager", "agent"} {
		if !strings.Contains(md, role) {
			t.Errorf("markdown не содержит роль %q", role)
		}
	}
	if !strings.Contains(md, "PARTIAL") {
		t.Error("markdown должен содержать статус PARTIAL")
	}
	if !strings.Contains(md, "quickstart") {
		t.Error("markdown должен содержать gap из consumer")
	}
	if !strings.Contains(md, "AGENTS.md") {
		t.Error("markdown должен содержать gap из agent")
	}
}

func TestRenderMarkdown_noJTBD(t *testing.T) {
	report := domain.Report{
		SchemaVersion: "1.0",
		Tool:          "rra-docs-another",
		Command:       "structure",
		Target:        domain.ReportTarget{Path: "/tmp/repo"},
	}
	md := renderMarkdown(report)
	if strings.Contains(md, "## JTBD") {
		t.Error("без JTBD секция ## JTBD не должна появляться")
	}
}

func TestRenderMarkdown_layersSorted(t *testing.T) {
	score := 100
	report := domain.Report{
		SchemaVersion: "1.0",
		Tool:          "rra-docs-another",
		Command:       "assess",
		Target:        domain.ReportTarget{Path: "/tmp/repo"},
		Layers: map[string]domain.LayerResult{
			"L6": {Name: "drift", Status: "pass", Score: &score},
			"L1": {Name: "readability", Status: "pass", Score: &score},
			"L3": {Name: "structure", Status: "pass", Score: &score},
		},
	}
	md := renderMarkdown(report)
	l1 := strings.Index(md, "L1")
	l3 := strings.Index(md, "L3")
	l6 := strings.Index(md, "L6")
	if l1 < 0 || l3 < 0 || l6 < 0 {
		t.Fatal("не все слои в markdown")
	}
	if !(l1 < l3 && l3 < l6) {
		t.Errorf("слои должны идти в порядке L1 < L3 < L6, got positions %d %d %d", l1, l3, l6)
	}
}
