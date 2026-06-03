package fitness

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

// ProcessFitness — голова слайса S5 (fitness, L5).
// Пайп: NewAuditTarget → store.ReadMarkdownDocs → Evaluate → buildReport.
// Валидация LLM-подключения (fail-fast по ключу/провайдеру) выполнена при
// сборке Deps в роутере (domain.NewLLMConfig). Фильтр cfg.Docs() применяется
// in-memory внутри Evaluate (один ReadMarkdownDocs на все слои).
func ProcessFitness(req domain.Request, deps Deps) (domain.Report, error) {
	target, err := domain.NewAuditTarget(req)
	if err != nil {
		return domain.Report{}, err
	}

	docs, err := deps.Store.ReadMarkdownDocs(target)
	if err != nil {
		return domain.Report{}, err
	}

	jtbdByRole, err := Evaluate(docs, deps.Config, deps.LLM)
	if err != nil {
		return domain.Report{}, err
	}

	return buildReport(target, req.Command, jtbdByRole), nil
}

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
