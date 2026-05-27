package structure

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

// ProcessStructure — голова-труба слайса S1 (structure): головной модуль (head.go),
// вызывается из register.go. Порядок:
// NewAuditTarget → NewConfig → store.ReadStructure → checkStructure → buildReport.
func ProcessStructure(req domain.Request, deps Deps) (domain.Report, error) {
	target, err := domain.NewAuditTarget(req)
	if err != nil {
		return domain.Report{}, err
	}

	cfg, err := domain.NewConfig(req)
	if err != nil {
		return domain.Report{}, err
	}

	structure, err := deps.Store.ReadStructure(target)
	if err != nil {
		return domain.Report{}, err
	}

	outcome := checkStructure(structure, cfg)

	parts := domain.ReportParts{
		Layers: []domain.LayerOutcome{outcome},
	}
	return buildReport(parts, target, req.Command), nil
}

// buildReport собирает Report из ReportParts, target и имени команды.
func buildReport(parts domain.ReportParts, target domain.AuditTarget, command string) domain.Report {
	layers := make(map[string]domain.LayerResult)
	var violations []domain.Violation

	for _, outcome := range parts.Layers {
		key := layerKey(outcome.Result.Name)
		layers[key] = outcome.Result
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

// layerKey возвращает ключ L3 для слоя по его имени.
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
