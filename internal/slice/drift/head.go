package drift

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

// ProcessDrift — голова-труба слайса S6 (drift, L6a).
// Пайп: NewAuditTarget → ReadStructure → extractClaims → NewDriftCheck →
// verifyClaims → buildClaimPromptSet → Judge → mergeSemanticFindings →
// NewDriftReport → buildDriftOutcome → buildReport.
// Решение --semantic (выбор реализации Judge) принимается в роутере, не здесь.
func ProcessDrift(req domain.Request, deps Deps) (domain.Report, error) {
	target, err := domain.NewAuditTarget(req)
	if err != nil {
		return domain.Report{}, err
	}

	cfg, err := domain.NewConfig(req)
	if err != nil {
		return domain.Report{}, err
	}

	structure, err := deps.Store.ReadStructure(target, cfg.Manifests())
	if err != nil {
		return domain.Report{}, err
	}

	outcome, err := Evaluate(structure, cfg, deps.Judge)
	if err != nil {
		return domain.Report{}, err
	}

	parts := domain.ReportParts{Layers: []domain.LayerOutcome{outcome}}
	return buildReport(parts, target, req.Command), nil
}

func buildReport(parts domain.ReportParts, target domain.AuditTarget, command string) domain.Report {
	layers := make(map[string]domain.LayerResult)
	var violations []domain.Violation

	for _, outcome := range parts.Layers {
		layers[layerKey(outcome.Result.Name)] = outcome.Result
		violations = append(violations, outcome.Violations...)
	}

	var commit *string
	if c := target.Commit(); c != "" {
		commit = &c
	}

	return domain.Report{
		SchemaVersion: "1.0",
		Tool:          "rra-docs-another",
		Command:       command,
		Target: domain.ReportTarget{
			Path:   target.Root(),
			Commit: commit,
		},
		Layers:     layers,
		Violations: violations,
	}
}

func layerKey(name string) string {
	switch name {
	case "structure":
		return "L3"
	case "readability":
		return "L1"
	case "style":
		return "L2"
	case "jtbd":
		return "L4"
	case "fitness":
		return "L5"
	case "drift":
		return "L6"
	default:
		return name
	}
}
