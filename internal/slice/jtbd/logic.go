// Package jtbd реализует слайс S3 — L4 JTBD-присутствие.
// Чистые функции: нет I/O, нет глобального состояния.
// Логика: matchHeadings строит индекс H1–H3, buildJTBDCard проверяет
// обязательные секции каждой роли против индекса.
package jtbd

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// headingIndex — нормализованный заголовок → "file:line"
type headingIndex map[string]string

// sectionSpec — обязательная секция для JTBD-роли.
// synonyms: список нормализованных синонимов; хотя бы один должен быть
// подстрокой нормализованного заголовка.
// critical: true → отсутствие даёт FAIL, false → PARTIAL.
type sectionSpec struct {
	synonyms []string
	critical bool
}

// consumerSpec — набор обязательных секций для одной JTBD-роли.
type consumerSpec struct {
	role     string
	sections []sectionSpec
}

// Словари синонимов для четырёх JTBD-потребителей.
// Первый синоним каждой секции — каноническое имя (появляется в gaps).
var (
	specMaintainer = consumerSpec{
		role: "maintainer",
		sections: []sectionSpec{
			{synonyms: []string{"архитектура", "architecture"}, critical: true},
			{synonyms: []string{"контрибьютить", "contributing"}, critical: true},
		},
	}
	specConsumer = consumerSpec{
		role: "consumer",
		sections: []sectionSpec{
			{synonyms: []string{"запуск", "quick start", "getting started", "install"}, critical: true},
			{synonyms: []string{"api", "апи", "endpoints"}, critical: false},
		},
	}
	specManager = consumerSpec{
		role: "manager",
		sections: []sectionSpec{
			{synonyms: []string{"умеет", "capabilities", "features", "overview"}, critical: true},
		},
	}
	specAgent = consumerSpec{
		role: "agent",
		sections: []sectionSpec{
			{synonyms: []string{"agents", "агент", "контекст"}, critical: true},
		},
	}
)

// matchHeadings нормализует H1–H3 из всех документов и возвращает headingIndex.
// Первое вхождение нормализованной формы побеждает при дублях.
func matchHeadings(docs []domain.MarkdownDoc, _ domain.Config) headingIndex {
	idx := make(headingIndex)
	for _, doc := range docs {
		for _, h := range doc.Headings {
			if h.Level > 3 {
				continue
			}
			norm := normalizeHeading(h.Text)
			if norm == "" {
				continue
			}
			if _, exists := idx[norm]; !exists {
				idx[norm] = fmt.Sprintf("%s:%d", doc.Path, h.Line)
			}
		}
	}
	return idx
}

// normalizeHeading приводит заголовок к нижнему регистру, заменяет
// не-буквы/не-цифры пробелами и схлопывает лишние пробелы.
func normalizeHeading(text string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		} else {
			sb.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}

// sectionPresent возвращает true, если хотя бы один нормализованный заголовок
// в индексе содержит один из синонимов секции как подстроку.
func sectionPresent(idx headingIndex, sec sectionSpec) bool {
	for norm := range idx {
		for _, syn := range sec.synonyms {
			if strings.Contains(norm, syn) {
				return true
			}
		}
	}
	return false
}

// buildJTBDCard проверяет обязательные секции роли против headingIndex.
//
// Статус:
//   - PASS — все секции найдены
//   - PARTIAL — все критичные найдены, часть некритичных отсутствует
//   - FAIL — хотя бы одна критичная секция отсутствует
//
// Score = (найденных / всего) × 100.
// Gaps — канонические имена отсутствующих секций.
func buildJTBDCard(idx headingIndex, spec consumerSpec) domain.JTBDResult {
	var gaps []string
	present := 0
	hasCriticalGap := false
	hasNonCriticalGap := false

	for _, sec := range spec.sections {
		if sectionPresent(idx, sec) {
			present++
		} else {
			gaps = append(gaps, sec.synonyms[0])
			if sec.critical {
				hasCriticalGap = true
			} else {
				hasNonCriticalGap = true
			}
		}
	}

	total := len(spec.sections)
	score := 100
	if total > 0 {
		score = present * 100 / total
	}

	var status string
	switch {
	case hasCriticalGap:
		status = "FAIL"
	case hasNonCriticalGap:
		status = "PARTIAL"
	default:
		status = "PASS"
	}

	if gaps == nil {
		gaps = []string{}
	}

	return domain.JTBDResult{
		Status: status,
		Score:  score,
		Gaps:   gaps,
	}
}
