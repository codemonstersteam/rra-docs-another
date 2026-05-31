package fitness

import (
	"strconv"
	"strings"
	"time"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

const defaultPromptBudget = 4096

// ── Бюджеты и пацинг — чистые листья (skill http-io «От curl к тестам») ───────

// estimateTokens — грубая оценка числа токенов в строке: байты/4.
// Используется пре-флайтом payload-бюджета ДО сетевого вызова, чтобы не
// отправлять заведомо лишний контекст (skill http-io → «Бюджет payload»).
func estimateTokens(s string) int { return len(s) / 4 }

// promptSetTokens — суммарная оценка токенов всех промптов набора (вход за команду).
func promptSetTokens(set domain.JTBDPromptSet) int {
	total := 0
	for _, p := range set.Prompts() {
		total += estimateTokens(p.Text())
	}
	return total
}

// overTokenBudget — предикат защитного лимита. limit<=0 отключает проверку.
func overTokenBudget(total, limit int) bool { return limit > 0 && total > limit }

// retryWait вычисляет паузу перед повтором transient-отказа (429).
// Приоритет — заголовок Retry-After (секунды); иначе экспоненциальный бэкофф
// base*2^attempt. Результат ограничен cap (skill http-io → «Пацинг»).
func retryWait(retryAfter string, attempt int, base, cap time.Duration) time.Duration {
	if secs, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && secs > 0 {
		d := time.Duration(secs) * time.Second
		if d > cap {
			return cap
		}
		return d
	}
	d := base
	for i := 0; i < attempt; i++ {
		d *= 2
		if d >= cap {
			return cap
		}
	}
	if d > cap {
		return cap
	}
	return d
}

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
