package readability

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

// ProcessReadability — голова-труба слайса S2 (readability): головной модуль (head.go),
// вызывается из register.go. Порядок:
// NewAuditTarget → NewConfig → store.ReadMarkdownDocs → scoreReadability → buildReport.
func ProcessReadability(req domain.Request, deps Deps) (domain.Report, error) {
	target, err := domain.NewAuditTarget(req)
	if err != nil {
		return domain.Report{}, err
	}

	cfg, err := domain.NewConfig(req)
	if err != nil {
		return domain.Report{}, err
	}

	docs, err := deps.Store.ReadMarkdownDocs(target)
	if err != nil {
		return domain.Report{}, err
	}

	outcome := Evaluate(docs, cfg)

	parts := domain.ReportParts{
		Layers: []domain.LayerOutcome{outcome},
	}
	return buildReport(parts, target, req.Command), nil
}

// buildReport собирает Report из ReportParts, target и имени команды.
// Скопировано дословно из S1 (слайс самодостаточен; консолидация — в S7 assess).
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

// layerKey возвращает ключ L1–L6 для слоя по его имени.
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
