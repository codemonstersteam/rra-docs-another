package jtbd_test

import (
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/jtbd"
)

// ── matchHeadings ─────────────────────────────────────────────────────────────

func TestMatchHeadings_happy(t *testing.T) {
	cfg := jtbd.ExportMakeConfig()
	docs := []domain.MarkdownDoc{
		makeDoc("docs/architecture.md", "Архитектура"),
		makeDoc("CONTRIBUTING.md", "Как контрибьютить"),
	}
	idx := jtbd.ExportMatchHeadings(docs, cfg)
	if len(idx) == 0 {
		t.Fatal("matchHeadings: индекс пустой, ожидали записи")
	}
	// Нормализованный "архитектура" должен быть в индексе.
	norm := jtbd.ExportNormalizeHeading("Архитектура")
	if _, ok := idx[norm]; !ok {
		t.Errorf("matchHeadings: ключ %q не найден в индексе", norm)
	}
}

func TestMatchHeadings_noHeadings(t *testing.T) {
	cfg := jtbd.ExportMakeConfig()
	docs := []domain.MarkdownDoc{
		{Path: "README.md", Lines: []string{"просто текст без заголовков"}, Headings: nil},
	}
	idx := jtbd.ExportMatchHeadings(docs, cfg)
	if len(idx) != 0 {
		t.Errorf("matchHeadings: ожидали пустой индекс, получили %d записей", len(idx))
	}
}

// ── buildJTBDCard ─────────────────────────────────────────────────────────────

func TestBuildJTBDCard_pass(t *testing.T) {
	cfg := jtbd.ExportMakeConfig()
	// maintainer требует архитектуру (critical) и contributing (critical).
	docs := []domain.MarkdownDoc{
		makeDoc("docs/architecture.md", "Архитектура"),
		makeDoc("CONTRIBUTING.md", "Как контрибьютить"),
	}
	idx := jtbd.ExportMatchHeadings(docs, cfg)
	result := jtbd.ExportBuildJTBDCard(idx, jtbd.ExportSpecMaintainer)

	if result.Status != "PASS" {
		t.Errorf("buildJTBDCard PASS: статус %q, ожидали PASS (gaps: %v)", result.Status, result.Gaps)
	}
	if result.Score != 100 {
		t.Errorf("buildJTBDCard PASS: score %d, ожидали 100", result.Score)
	}
	if len(result.Gaps) != 0 {
		t.Errorf("buildJTBDCard PASS: gaps не пустые: %v", result.Gaps)
	}
}

func TestBuildJTBDCard_partial(t *testing.T) {
	cfg := jtbd.ExportMakeConfig()
	// consumer: "запуск" (critical) и "api" (non-critical).
	// Дадим только "запуск" → PARTIAL (non-critical api отсутствует).
	docs := []domain.MarkdownDoc{
		makeDoc("README.md", "Запуск"),
	}
	idx := jtbd.ExportMatchHeadings(docs, cfg)
	result := jtbd.ExportBuildJTBDCard(idx, jtbd.ExportSpecConsumer)

	if result.Status != "PARTIAL" {
		t.Errorf("buildJTBDCard PARTIAL: статус %q, ожидали PARTIAL (gaps: %v)", result.Status, result.Gaps)
	}
	if len(result.Gaps) == 0 {
		t.Error("buildJTBDCard PARTIAL: gaps должны содержать пропущенную некритичную секцию")
	}
}

func TestBuildJTBDCard_fail(t *testing.T) {
	cfg := jtbd.ExportMakeConfig()
	// maintainer: архитектура (critical) и contributing (critical).
	// Дадим только contributing → FAIL (архитектура отсутствует — критичная).
	docs := []domain.MarkdownDoc{
		makeDoc("CONTRIBUTING.md", "Как контрибьютить"),
	}
	idx := jtbd.ExportMatchHeadings(docs, cfg)
	result := jtbd.ExportBuildJTBDCard(idx, jtbd.ExportSpecMaintainer)

	if result.Status != "FAIL" {
		t.Errorf("buildJTBDCard FAIL: статус %q, ожидали FAIL (gaps: %v)", result.Status, result.Gaps)
	}
	if len(result.Gaps) == 0 {
		t.Error("buildJTBDCard FAIL: gaps должны содержать критичную отсутствующую секцию")
	}
}
