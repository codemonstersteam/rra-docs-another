// testhelpers_test.go — вспомогательные конструкторы только для тестов.
package jtbd_test

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

// makeDoc создаёт MarkdownDoc с ATX-заголовками из headingTexts (H1, line N+1).
func makeDoc(path string, headingTexts ...string) domain.MarkdownDoc {
	var headings []domain.Heading
	for i, text := range headingTexts {
		headings = append(headings, domain.Heading{Level: 1, Text: text, Line: i + 1})
	}
	return domain.MarkdownDoc{Path: path, Headings: headings}
}
