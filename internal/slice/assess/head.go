package assess

import (
	"github.com/codemonstersteam/rra-docs-another/internal/domain"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/drift"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/fitness"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/jtbd"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/readability"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/structure"
)

// ProcessAssess — голова-интегратор слайса S7 (assess).
// Пайп: NewAuditTarget → NewConfig → ReadStructure (1×) → layersUpTo →
// листья Evaluate по плану → hasDocs-гейт → условный L5 → mergeOutcomes
// (кэп L5 статикой L4).
// LLMConfig резолвится в голове по ветке L5 (нет ключа → ErrLLMUnavailable → код 2).
func ProcessAssess(req domain.Request, deps Deps) (domain.Report, error) {
	target, err := domain.NewAuditTarget(req)
	if err != nil {
		return domain.Report{}, err
	}

	cfg, err := domain.NewConfig(req)
	if err != nil {
		return domain.Report{}, err
	}

	s, err := deps.Store.ReadStructure(target, cfg.Manifests())
	if err != nil {
		return domain.Report{}, err
	}

	plan := layersUpTo(req.UpTo)
	var out layerOutcomes

	if plan.L1 {
		out.l1 = readability.Evaluate(s.Docs, cfg)
	}
	if plan.L3 {
		out.l3 = structure.Evaluate(s, cfg)
	}
	if plan.L4 {
		out.l4 = jtbd.Evaluate(s.Docs, cfg)
	}
	if plan.L6 {
		out.l6, err = drift.Evaluate(s, cfg, deps.Judge)
		if err != nil {
			return domain.Report{}, err
		}
	}
	if plan.L5 && hasDocs(s.Docs) {
		llmCfg, llmErr := domain.NewLLMConfig(req, cfg)
		if llmErr != nil {
			return domain.Report{}, llmErr
		}
		llm := fitness.NewLLMClient(llmCfg, cfg.LLMCallDelayMs(), cfg.LLMTokenBudget(), cfg.LLMMaxRetries())
		out.l5, err = fitness.Evaluate(s.Docs, cfg, llm)
		if err != nil {
			return domain.Report{}, err
		}
	}

	return mergeOutcomes(plan, target, out), nil
}
