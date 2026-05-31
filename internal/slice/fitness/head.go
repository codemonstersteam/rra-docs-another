package fitness

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

// ProcessFitness — голова слайса S5 (fitness, L5).
// Пайп: NewAuditTarget → store.ReadMarkdownDocsByList → buildJTBDPromptSet →
// llm.Ask → scoreFitness → buildReport. Валидация LLM-подключения (fail-fast по
// ключу/провайдеру) выполнена при сборке Deps в роутере (domain.NewLLMConfig).
func ProcessFitness(req domain.Request, deps Deps) (domain.Report, error) {
	target, err := domain.NewAuditTarget(req)
	if err != nil {
		return domain.Report{}, err
	}

	var docs []domain.MarkdownDoc
	if list := deps.Config.Docs(); len(list) > 0 {
		docs, err = deps.Store.ReadMarkdownDocsByList(target, list)
	} else {
		docs, err = deps.Store.ReadMarkdownDocs(target)
	}
	if err != nil {
		return domain.Report{}, err
	}

	promptSet := buildJTBDPromptSet(docs, deps.Config)

	verdicts, err := deps.LLM.Ask(promptSet)
	if err != nil {
		return domain.Report{}, err
	}

	results := scoreFitness(verdicts)

	jtbdByRole := make(map[string]domain.JTBDResult, len(verdicts))
	for i, v := range verdicts {
		jtbdByRole[v.Consumer] = results[i]
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
