package assess

import (
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// ── layersUpTo ────────────────────────────────────────────────────────────────

func TestLayersUpTo_default(t *testing.T) {
	plan := layersUpTo("")
	if !plan.L1 || !plan.L3 || !plan.L4 || !plan.L5 || !plan.L6 {
		t.Errorf("default plan должен включать все слои, got %+v", plan)
	}
}

func TestLayersUpTo_L6(t *testing.T) {
	plan := layersUpTo("L6")
	if !plan.L1 || !plan.L3 || !plan.L4 || !plan.L5 || !plan.L6 {
		t.Errorf("L6 должен включать все слои, got %+v", plan)
	}
}

func TestLayersUpTo_L4(t *testing.T) {
	plan := layersUpTo("L4")
	if !plan.L1 || !plan.L3 || !plan.L4 {
		t.Errorf("L4 должен включать L1/L3/L4, got %+v", plan)
	}
	if plan.L5 || plan.L6 {
		t.Errorf("L4 не должен включать L5/L6, got %+v", plan)
	}
}

func TestLayersUpTo_L1(t *testing.T) {
	plan := layersUpTo("L1")
	if !plan.L1 {
		t.Errorf("L1 должен включать L1, got %+v", plan)
	}
	if plan.L3 || plan.L4 || plan.L5 || plan.L6 {
		t.Errorf("L1 не должен включать другие слои, got %+v", plan)
	}
}

// ── shortCircuit ──────────────────────────────────────────────────────────────

func TestShortCircuit_noFail(t *testing.T) {
	l4 := map[string]domain.JTBDResult{
		"maintainer": {Status: "PASS"},
		"consumer":   {Status: "PARTIAL"},
	}
	if shortCircuit(l4) {
		t.Error("без FAIL shortCircuit должен возвращать false")
	}
}

func TestShortCircuit_hasFail(t *testing.T) {
	l4 := map[string]domain.JTBDResult{
		"maintainer": {Status: "PASS"},
		"consumer":   {Status: "FAIL"},
	}
	if !shortCircuit(l4) {
		t.Error("при FAIL shortCircuit должен возвращать true")
	}
}

func TestShortCircuit_empty(t *testing.T) {
	if shortCircuit(nil) {
		t.Error("пустая карта → false")
	}
	if shortCircuit(map[string]domain.JTBDResult{}) {
		t.Error("пустая карта → false")
	}
}

// ── mergeOutcomes ─────────────────────────────────────────────────────────────

func makeTarget(t *testing.T) domain.AuditTarget {
	t.Helper()
	// Используем текущую директорию как путь (существует).
	target, err := domain.NewAuditTarget(domain.Request{Path: "."})
	if err != nil {
		t.Fatalf("NewAuditTarget: %v", err)
	}
	return target
}

func TestMergeOutcomes_l5ExecutedJTBDFromL5(t *testing.T) {
	plan := layersUpTo("L6")
	target := makeTarget(t)

	l1score := 80
	l3score := 100
	l6score := 100
	out := layerOutcomes{
		l1: domain.LayerOutcome{Result: domain.LayerResult{Name: "readability", Status: "pass", Score: &l1score}},
		l3: domain.LayerOutcome{Result: domain.LayerResult{Name: "structure", Status: "pass", Score: &l3score}},
		l4: map[string]domain.JTBDResult{
			"maintainer": {Status: "PASS", Score: 80},
		},
		l5: map[string]domain.JTBDResult{
			"maintainer": {Status: "PASS", Score: 95},
		},
		l6: domain.LayerOutcome{Result: domain.LayerResult{Name: "drift", Status: "pass", Score: &l6score}},
	}

	report := mergeOutcomes(plan, target, out)

	if report.Command != "assess" {
		t.Errorf("command = %q, want assess", report.Command)
	}
	if report.Layers["L1"].Status != "pass" {
		t.Errorf("L1 status = %q", report.Layers["L1"].Status)
	}
	if report.Layers["L3"].Status != "pass" {
		t.Errorf("L3 status = %q", report.Layers["L3"].Status)
	}
	if report.Layers["L6"].Status != "pass" {
		t.Errorf("L6 status = %q", report.Layers["L6"].Status)
	}
	// jtbd из L5 (score 95, не 80)
	if r := report.JTBD["maintainer"]; r.Score != 95 {
		t.Errorf("jtbd maintainer score = %d, want 95 (из L5)", r.Score)
	}
}

func TestMergeOutcomes_l5NotExecutedJTBDFromL4(t *testing.T) {
	plan := layersUpTo("L4")
	target := makeTarget(t)

	l1score := 70
	l3score := 100
	out := layerOutcomes{
		l1: domain.LayerOutcome{Result: domain.LayerResult{Name: "readability", Status: "pass", Score: &l1score}},
		l3: domain.LayerOutcome{Result: domain.LayerResult{Name: "structure", Status: "pass", Score: &l3score}},
		l4: map[string]domain.JTBDResult{
			"consumer": {Status: "PARTIAL", Score: 50},
		},
	}

	report := mergeOutcomes(plan, target, out)

	// L5, L6 вне плана → skipped
	if report.Layers["L6"].Status != "skipped" {
		t.Errorf("L6 должен быть skipped, got %q", report.Layers["L6"].Status)
	}
	// jtbd из L4
	if r := report.JTBD["consumer"]; r.Score != 50 {
		t.Errorf("jtbd consumer score = %d, want 50 (из L4)", r.Score)
	}
}

func TestMergeOutcomes_violationsUnion(t *testing.T) {
	plan := layersUpTo("L6")
	target := makeTarget(t)

	l1score := 40
	l3score := 50
	l6score := 50
	out := layerOutcomes{
		l1: domain.LayerOutcome{
			Result:     domain.LayerResult{Name: "readability", Status: "warn", Score: &l1score},
			Violations: []domain.Violation{{Code: "low_readability", Layer: "L1", Severity: "warning"}},
		},
		l3: domain.LayerOutcome{
			Result:     domain.LayerResult{Name: "structure", Status: "fail", Score: &l3score},
			Violations: []domain.Violation{{Code: "missing_required_file", Layer: "L3", Severity: "blocker"}},
		},
		l6: domain.LayerOutcome{
			Result:     domain.LayerResult{Name: "drift", Status: "fail", Score: &l6score},
			Violations: []domain.Violation{{Code: "doc_drift", Layer: "L6", Severity: "blocker"}},
		},
	}

	report := mergeOutcomes(plan, target, out)

	if len(report.Violations) != 3 {
		t.Errorf("violations count = %d, want 3", len(report.Violations))
	}
}
