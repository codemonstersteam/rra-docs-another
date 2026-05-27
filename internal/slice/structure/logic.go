// Package structure реализует слайс S1 — L3 структурная полнота.
// Чистые функции: нет I/O, нет глобального состояния.
package structure

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// checkReadmePresent проверяет наличие README.md в корне репозитория.
// Failure: Violation{severity:blocker}.
func checkReadmePresent(structure domain.RepoStructure) []domain.Violation {
	for _, f := range structure.Files {
		// README.md прямо в корне (без поддиректорий).
		base := filepath.Base(f)
		dir := filepath.Dir(f)
		if dir == "." && strings.EqualFold(base, "readme.md") {
			return nil
		}
	}
	return []domain.Violation{{
		Code:     "missing_readme",
		Layer:    "L3",
		Severity: "blocker",
		File:     "README.md",
		Message:  "README.md отсутствует в корне репозитория",
	}}
}

// checkLinksResolve проверяет, что все Markdown-ссылки на локальные файлы резолвятся.
// Failure: Violation{severity:blocker} для каждой битой ссылки.
func checkLinksResolve(structure domain.RepoStructure) []domain.Violation {
	fileSet := make(map[string]struct{}, len(structure.Files))
	for _, f := range structure.Files {
		// Нормализуем путь: разделитель на '/'.
		fileSet[filepath.ToSlash(f)] = struct{}{}
	}

	var violations []domain.Violation
	for _, doc := range structure.Docs {
		docDir := filepath.Dir(doc.Path)
		for i, line := range doc.Lines {
			links := extractLocalLinks(line)
			for _, link := range links {
				resolved := resolveLink(docDir, link)
				if _, ok := fileSet[resolved]; !ok {
					lineNum := i + 1
					violations = append(violations, domain.Violation{
						Code:     "broken_link",
						Layer:    "L3",
						Severity: "blocker",
						File:     filepath.ToSlash(doc.Path),
						Line:     &lineNum,
						Message:  fmt.Sprintf("битая ссылка: %s (→ %s не найден)", link, resolved),
					})
				}
			}
		}
	}
	return violations
}

// extractLocalLinks извлекает URL из Markdown-ссылок вида [text](url).
// Возвращает только локальные (без схемы, без #-только якорей).
func extractLocalLinks(line string) []string {
	var links []string
	rest := line
	for {
		idx := strings.Index(rest, "](")
		if idx < 0 {
			break
		}
		after := rest[idx+2:]
		end := strings.Index(after, ")")
		if end < 0 {
			break
		}
		url := after[:end]
		rest = after[end+1:]
		// Пропускаем: внешние (http/https/mailto/ftp), пустые, якоря.
		if url == "" || strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") ||
			strings.HasPrefix(url, "mailto:") || strings.HasPrefix(url, "ftp://") ||
			strings.HasPrefix(url, "#") {
			continue
		}
		// Убираем якорную часть (#).
		if i := strings.Index(url, "#"); i >= 0 {
			url = url[:i]
		}
		if url == "" {
			continue
		}
		links = append(links, url)
	}
	return links
}

// resolveLink разрешает ссылку link относительно директории docDir.
func resolveLink(docDir, link string) string {
	if filepath.IsAbs(link) {
		return filepath.ToSlash(link[1:])
	}
	resolved := filepath.Join(docDir, link)
	return filepath.ToSlash(resolved)
}

// checkDocDrift выявляет документы, которые не обновлялись дольше threshold дней,
// при условии что в репо есть код, изменявшийся за это время.
// Failure: Violation{severity:warning}.
func checkDocDrift(structure domain.RepoStructure, cfg domain.Config) []domain.Violation {
	threshold := time.Duration(cfg.DriftThresholdDays()) * 24 * time.Hour
	now := time.Now()

	// Ищем самое свежее время модификации среди НЕ-doc файлов.
	latestCode := time.Time{}
	for _, f := range structure.Files {
		if strings.HasSuffix(strings.ToLower(f), ".md") {
			continue
		}
		if t, ok := structure.MTimes[f]; ok && t.After(latestCode) {
			latestCode = t
		}
	}

	var violations []domain.Violation
	for _, doc := range structure.Docs {
		docMtime, ok := structure.MTimes[doc.Path]
		if !ok {
			continue
		}
		age := now.Sub(docMtime)
		if age > threshold && !latestCode.IsZero() && latestCode.After(docMtime) {
			violations = append(violations, domain.Violation{
				Code:     "doc_drift",
				Layer:    "L3",
				Severity: "warning",
				File:     filepath.ToSlash(doc.Path),
				Message:  fmt.Sprintf("документ не обновлялся %d дней (порог %d)", int(age.Hours()/24), cfg.DriftThresholdDays()),
			})
		}
	}
	return violations
}

// checkStructure агрегирует под-проверки L3 в LayerOutcome.
// Status=fail при наличии blocker; warn если только warning; иначе pass.
// Score — доля пройденных проверок (0–100).
func checkStructure(structure domain.RepoStructure, cfg domain.Config) domain.LayerOutcome {
	readme := checkReadmePresent(structure)
	links := checkLinksResolve(structure)
	drift := checkDocDrift(structure, cfg)

	all := make([]domain.Violation, 0, len(readme)+len(links)+len(drift))
	all = append(all, readme...)
	all = append(all, links...)
	all = append(all, drift...)

	// Score — доля пройденных из трёх проверок. README и ссылки могут дать blocker;
	// drift — только warning, в знаменателе считается всегда пройденной.
	const total = 3
	failed := 0
	if hasBlocker(readme) {
		failed++
	}
	if hasBlocker(links) {
		failed++
	}
	score := (total - failed) * 100 / total

	status := "pass"
	switch {
	case hasBlocker(all):
		status = "fail"
	case len(all) > 0:
		status = "warn"
	}

	return domain.LayerOutcome{
		Result: domain.LayerResult{
			Name:    "structure",
			Status:  status,
			Score:   &score,
			Summary: buildSummary(status, all),
		},
		Violations: all,
	}
}

func hasBlocker(vs []domain.Violation) bool {
	for _, v := range vs {
		if v.Severity == "blocker" {
			return true
		}
	}
	return false
}

func buildSummary(status string, vs []domain.Violation) string {
	switch status {
	case "pass":
		return "структурная полнота в порядке"
	case "fail":
		blockers := 0
		for _, v := range vs {
			if v.Severity == "blocker" {
				blockers++
			}
		}
		return fmt.Sprintf("структурные нарушения: %d blocker(s)", blockers)
	case "warn":
		return fmt.Sprintf("предупреждения: %d warning(s)", len(vs))
	default:
		return ""
	}
}
