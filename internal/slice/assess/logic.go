// Package assess реализует слайс S7 — полный пайплайн аудита L1/L3/L4/L5/L6a.
// Порядок: дешёвое-первым; L5 условно (plan≥L5 && !shortCircuit(L4)).
// Новых I/O нет. Чистые листья — Evaluate каждого слайса из E15.
package assess

import (
	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// LayerPlan — множество слоёв, запланированных к исполнению.
// Формируется из --up-to; L2 отсутствует (S4 в TBD).
type LayerPlan struct {
	L1, L3, L4, L5, L6 bool
}

// layerOutcomes — аккумулятор исполненных оценок.
// Поля заполняются в голове по плану; mergeOutcomes читает через план.
type layerOutcomes struct {
	l1 domain.LayerOutcome
	l3 domain.LayerOutcome
	l4 map[string]domain.JTBDResult
	l5 map[string]domain.JTBDResult
	l6 domain.LayerOutcome
}

// layersUpTo строит план из флага --up-to.
// upTo ∈ {"", "L1".."L6"}; "" = L6 (все). L2 не имеет слоя — если upTo=L2,
// в план входит только L1.
func layersUpTo(upTo string) LayerPlan {
	if upTo == "" {
		upTo = "L6"
	}
	ord := map[string]int{
		"L1": 1, "L2": 2, "L3": 3, "L4": 4, "L5": 5, "L6": 6,
	}
	limit := ord[upTo]
	return LayerPlan{
		L1: ord["L1"] <= limit,
		L3: ord["L3"] <= limit,
		L4: ord["L4"] <= limit,
		L5: ord["L5"] <= limit,
		L6: ord["L6"] <= limit,
	}
}

// shortCircuit возвращает true, если хотя бы один L4-результат FAIL.
// При FAIL L4 → L5/LLM не запускается.
// Пустая карта (L4 вне плана) → false.
func shortCircuit(l4 map[string]domain.JTBDResult) bool {
	for _, r := range l4 {
		if r.Status == "FAIL" {
			return true
		}
	}
	return false
}

// mergeOutcomes собирает единый Report из плана и исполненных оценок.
//
// layers: L1/L3/L6 — фактический результат если в плане, иначе skipped-маркер.
// jtbd: из L5 если исполнялся, иначе из L4; отсутствует если план < L4.
// violations: конкатенация из всех исполненных слоёв.
func mergeOutcomes(plan LayerPlan, target domain.AuditTarget, out layerOutcomes) domain.Report {
	layers := make(map[string]domain.LayerResult)
	var violations []domain.Violation

	if plan.L1 {
		layers["L1"] = out.l1.Result
		violations = append(violations, out.l1.Violations...)
	} else {
		layers["L1"] = domain.LayerResult{Name: "readability", Status: "skipped"}
	}

	if plan.L3 {
		layers["L3"] = out.l3.Result
		violations = append(violations, out.l3.Violations...)
	} else {
		layers["L3"] = domain.LayerResult{Name: "structure", Status: "skipped"}
	}

	if plan.L6 {
		layers["L6"] = out.l6.Result
		violations = append(violations, out.l6.Violations...)
	} else {
		layers["L6"] = domain.LayerResult{Name: "drift", Status: "skipped"}
	}

	var jtbd map[string]domain.JTBDResult
	switch {
	case out.l5 != nil:
		jtbd = out.l5
	case out.l4 != nil:
		jtbd = out.l4
	}

	var commit *string
	if c := target.Commit(); c != "" {
		commit = &c
	}

	return domain.Report{
		SchemaVersion: "1.0",
		Tool:          "rra-docs-another",
		Command:       "assess",
		Target: domain.ReportTarget{
			Path:   target.Root(),
			Commit: commit,
		},
		Layers:     layers,
		JTBD:       jtbd,
		Violations: violations,
	}
}
