package jtbd

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

// ProcessJTBD — голова-труба слайса S3 (jtbd).
// Порядок: NewAuditTarget → NewConfig → store.ReadMarkdownDocs →
// matchHeadings → buildJTBDCard по каждой роли из конфига → buildReport.
func ProcessJTBD(req domain.Request, deps Deps) (domain.Report, error) {
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

	idx := matchHeadings(docs, cfg)

	consumers := cfg.JTBDSpec().Consumers()
	jtbdByRole := make(map[string]domain.JTBDResult, len(consumers))
	for _, consumer := range consumers {
		jtbdByRole[consumer.Role()] = buildJTBDCard(idx, consumer)
	}

	return buildReport(target, req.Command, jtbdByRole), nil
}

// buildReport собирает Report с JTBD-секцией.
// layers.L4 слайс jtbd не заполняет — это делает S7 assess.
func buildReport(target domain.AuditTarget, command string, jtbdByRole map[string]domain.JTBDResult) domain.Report {
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
		JTBD: jtbdByRole,
	}
}
