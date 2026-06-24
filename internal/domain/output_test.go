package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// ── NewReportOutput: формула 1 happy + Σ ветки (формат × назначение) ─────────

func TestNewReportOutput_jsonStdout_happy(t *testing.T) {
	report := domain.Report{Command: "structure", Target: domain.ReportTarget{Path: "/r"}}
	out, err := domain.NewReportOutput(report, domain.Request{Format: "json", Out: "-"})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(out.Content()), "{") {
		t.Errorf("json-контент должен начинаться с {: %q", out.Content())
	}
	if out.Dest().IsFile() {
		t.Error("Out=- → назначение stdout, не файл")
	}
}

func TestNewReportOutput_emptyFormatDefaultsJSON(t *testing.T) {
	out, err := domain.NewReportOutput(domain.Report{Command: "structure"}, domain.Request{Out: "-"})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(out.Content()), "{") {
		t.Errorf("пустой формат → json, got %q", out.Content())
	}
}

func TestNewReportOutput_mdToFile(t *testing.T) {
	report := domain.Report{Command: "assess", Target: domain.ReportTarget{Path: "/r"}}
	out, err := domain.NewReportOutput(report, domain.Request{Format: "md", Out: "/tmp/report.md"})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if !strings.Contains(out.Content(), "# rra-docs-another") {
		t.Errorf("md-контент должен содержать заголовок: %q", out.Content())
	}
	if !out.Dest().IsFile() || out.Dest().Path() != "/tmp/report.md" {
		t.Errorf("Out=путь → файл с этим путём, got file=%v path=%q", out.Dest().IsFile(), out.Dest().Path())
	}
}

func TestNewReportOutput_unknownFormat(t *testing.T) {
	_, err := domain.NewReportOutput(domain.Report{}, domain.Request{Format: "xml"})
	if !errors.Is(err, domain.ErrUnknownFormat) {
		t.Errorf("ожидали ErrUnknownFormat, got %v", err)
	}
}

// ── renderMarkdown (через NewReportOutput format=md, чёрный ящик) ─────────────

func TestRenderMarkdown_jtbdSection(t *testing.T) {
	report := domain.Report{
		Command: "assess",
		Target:  domain.ReportTarget{Path: "/tmp/repo"},
		JTBD: map[string]domain.JTBDResult{
			"maintainer": {Status: "PASS", Score: 90, Gaps: []string{}},
			"consumer":   {Status: "PARTIAL", Score: 60, Gaps: []string{"quickstart"}},
			"agent":      {Status: "FAIL", Score: 20, Gaps: []string{"AGENTS.md"}},
		},
	}
	out, err := domain.NewReportOutput(report, domain.Request{Format: "md"})
	if err != nil {
		t.Fatal(err)
	}
	md := out.Content()
	for _, want := range []string{"## JTBD", "maintainer", "consumer", "agent", "PARTIAL", "quickstart", "AGENTS.md"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown не содержит %q", want)
		}
	}
}

func TestRenderMarkdown_noJTBD(t *testing.T) {
	out, _ := domain.NewReportOutput(domain.Report{Command: "structure"}, domain.Request{Format: "md"})
	if strings.Contains(out.Content(), "## JTBD") {
		t.Error("без JTBD секция ## JTBD не должна появляться")
	}
}

func TestRenderMarkdown_layersSorted(t *testing.T) {
	score := 100
	report := domain.Report{
		Command: "assess",
		Target:  domain.ReportTarget{Path: "/tmp/repo"},
		Layers: map[string]domain.LayerResult{
			"L6": {Name: "drift", Status: "pass", Score: &score},
			"L1": {Name: "readability", Status: "pass", Score: &score},
			"L3": {Name: "structure", Status: "pass", Score: &score},
		},
	}
	out, _ := domain.NewReportOutput(report, domain.Request{Format: "md"})
	md := out.Content()
	l1, l3, l6 := strings.Index(md, "L1"), strings.Index(md, "L3"), strings.Index(md, "L6")
	if l1 < 0 || l3 < 0 || l6 < 0 {
		t.Fatal("не все слои в markdown")
	}
	if !(l1 < l3 && l3 < l6) {
		t.Errorf("порядок L1<L3<L6, got %d %d %d", l1, l3, l6)
	}
}
