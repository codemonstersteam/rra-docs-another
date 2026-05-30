package fitness

import (
	"strings"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

const defaultPromptBudget = 4096

var jtbdRoles = []string{"maintainer", "consumer", "manager", "agent"}

// buildJTBDPromptSet строит набор из 4 промптов по ролям.
// Промпты берутся из cfg, к каждому добавляется содержимое docs.
func buildJTBDPromptSet(docs []domain.MarkdownDoc, cfg domain.Config) domain.JTBDPromptSet {
	docsContent := formatDocs(docs)
	prompts := make([]domain.JTBDPrompt, 0, len(jtbdRoles))
	for _, role := range jtbdRoles {
		text := cfg.LLMPrompt(role)
		if docsContent != "" {
			text += "\n\nДокументация репозитория:\n" + docsContent
		}
		prompts = append(prompts, domain.NewJTBDPrompt(role, text, defaultPromptBudget))
	}
	return domain.NewJTBDPromptSet(prompts)
}

// scoreFitness нормализует сырые вердикты в JTBDResult.
// Некорректный статус → консервативный PARTIAL.
func scoreFitness(verdicts []domain.LLMVerdict) []domain.JTBDResult {
	results := make([]domain.JTBDResult, 0, len(verdicts))
	for _, v := range verdicts {
		status := v.RawStatus
		if status != "PASS" && status != "FAIL" && status != "PARTIAL" {
			status = "PARTIAL"
		}
		gaps := v.RawGaps
		if gaps == nil {
			gaps = []string{}
		}
		results = append(results, domain.JTBDResult{
			Status: status,
			Score:  v.RawScore,
			Gaps:   gaps,
		})
	}
	return results
}

func formatDocs(docs []domain.MarkdownDoc) string {
	if len(docs) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, doc := range docs {
		sb.WriteString("## ")
		sb.WriteString(doc.Path)
		sb.WriteString("\n")
		sb.WriteString(strings.Join(doc.Lines, "\n"))
		sb.WriteString("\n\n")
	}
	return sb.String()
}
