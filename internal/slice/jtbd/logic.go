// Package jtbd реализует слайс S3 — L4 JTBD-присутствие.
// Чистые функции: нет I/O, нет глобального состояния.
// Логика: matchHeadings строит индекс H1–H3, buildJTBDCard проверяет
// обязательные секции каждой роли против индекса. Словари секций приходят
// из конфига (domain.JTBDSpec), не хардкодятся.
package jtbd

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// headingIndex — нормализованный заголовок → "file:line"
type headingIndex map[string]string

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
func sectionPresent(idx headingIndex, sec domain.JTBDSection) bool {
	for norm := range idx {
		for _, syn := range sec.Synonyms() {
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
func buildJTBDCard(idx headingIndex, consumer domain.JTBDConsumer) domain.JTBDResult {
	var gaps []string
	present := 0
	hasCriticalGap := false
	hasNonCriticalGap := false

	sections := consumer.Sections()
	for _, sec := range sections {
		if sectionPresent(idx, sec) {
			present++
		} else {
			gaps = append(gaps, sec.Name())
			if sec.Critical() {
				hasCriticalGap = true
			} else {
				hasNonCriticalGap = true
			}
		}
	}

	total := len(sections)
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
