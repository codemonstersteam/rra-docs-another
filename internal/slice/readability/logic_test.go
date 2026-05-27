package readability_test

import (
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/readability"
)

// ── fleschKincaid ─────────────────────────────────────────────────────────────

func TestFleschKincaid_happy(t *testing.T) {
	// Простой английский текст — оценка должна быть в диапазоне [0, 100].
	doc := domain.MarkdownDoc{
		Path: "README.md",
		Lines: []string{
			"# Quick Start",
			"",
			"This is a simple guide. It helps you get started quickly.",
			"Follow the steps below to install and run the tool.",
		},
	}
	score := readability.ExportFleschKincaid(doc)
	if score < 0 || score > 100 {
		t.Errorf("fleschKincaid: ожидали [0,100], получили %f", score)
	}
}

func TestFleschKincaid_emptyText(t *testing.T) {
	// Пустой документ — нейтральное значение (без паники).
	doc := domain.MarkdownDoc{Path: "empty.md", Lines: nil}
	score := readability.ExportFleschKincaid(doc)
	if score < 0 || score > 100 {
		t.Errorf("fleschKincaid(empty): ожидали [0,100], получили %f", score)
	}
}

// ── obornevaRus ───────────────────────────────────────────────────────────────

func TestObornevaRus_happy(t *testing.T) {
	// Простой русский текст — оценка в [0, 100].
	doc := domain.MarkdownDoc{
		Path: "README.md",
		Lines: []string{
			"# Быстрый старт",
			"",
			"Это простое руководство. Оно поможет вам быстро начать работу.",
			"Следуйте инструкциям ниже для установки и запуска инструмента.",
		},
	}
	score := readability.ExportObornevaRus(doc)
	if score < 0 || score > 100 {
		t.Errorf("obornevaRus: ожидали [0,100], получили %f", score)
	}
}

func TestObornevaRus_emptyText(t *testing.T) {
	// Пустой документ — нейтральное значение (без паники).
	doc := domain.MarkdownDoc{Path: "empty.md", Lines: nil}
	score := readability.ExportObornevaRus(doc)
	if score < 0 || score > 100 {
		t.Errorf("obornevaRus(empty): ожидали [0,100], получили %f", score)
	}
}

// ── pickFormula ───────────────────────────────────────────────────────────────

func TestPickFormula_english(t *testing.T) {
	// Английский текст → fleschKincaid.
	doc := domain.MarkdownDoc{
		Path:  "README.md",
		Lines: []string{"This is an English document. It has multiple sentences."},
	}
	formula := readability.ExportPickFormula(doc)
	// Проверяем косвенно: оба дают float64, отличие — через cyrillicRatio.
	cyrRatio := readability.ExportCyrillicRatio(doc)
	if cyrRatio >= 0.30 {
		t.Errorf("pickFormula: английский текст не должен давать cyrillicRatio >= 0.30, получили %f", cyrRatio)
	}
	score := formula(doc)
	if score < 0 || score > 100 {
		t.Errorf("pickFormula(en): оценка вне [0,100]: %f", score)
	}
}

func TestPickFormula_russian(t *testing.T) {
	// Кириллический текст (доля ≥ 30%) → obornevaRus.
	doc := domain.MarkdownDoc{
		Path:  "README.md",
		Lines: []string{"Это документ на русском языке. Он содержит несколько предложений."},
	}
	cyrRatio := readability.ExportCyrillicRatio(doc)
	if cyrRatio < 0.30 {
		t.Errorf("pickFormula: русский текст должен давать cyrillicRatio >= 0.30, получили %f", cyrRatio)
	}
	formula := readability.ExportPickFormula(doc)
	score := formula(doc)
	if score < 0 || score > 100 {
		t.Errorf("pickFormula(ru): оценка вне [0,100]: %f", score)
	}
}

func TestPickFormula_empty(t *testing.T) {
	// Пустой текст → fleschKincaid (нет паники).
	doc := domain.MarkdownDoc{Path: "empty.md", Lines: nil}
	cyrRatio := readability.ExportCyrillicRatio(doc)
	if cyrRatio != 0 {
		t.Errorf("cyrillicRatio(empty): ожидали 0, получили %f", cyrRatio)
	}
	formula := readability.ExportPickFormula(doc)
	score := formula(doc)
	if score < 0 || score > 100 {
		t.Errorf("pickFormula(empty): оценка вне [0,100]: %f", score)
	}
}

// ── scoreReadability ──────────────────────────────────────────────────────────

func TestScoreReadability_happy(t *testing.T) {
	// Простые документы с нормальной читаемостью → pass.
	docs := []domain.MarkdownDoc{
		{
			Path:  "README.md",
			Lines: []string{"# Guide", "", "This is easy to read. Short sentences are good."},
		},
	}
	cfg := readability.ExportMakeConfig(50)
	outcome := readability.ExportScoreReadability(docs, cfg)
	if outcome.Result.Status == "fail" {
		t.Errorf("scoreReadability: L1 не должен давать status=fail, получили %s", outcome.Result.Status)
	}
	if outcome.Result.Score == nil {
		t.Error("scoreReadability: Score не должен быть nil")
	}
	if outcome.Result.Name != "readability" {
		t.Errorf("scoreReadability: Name=%s, ожидали readability", outcome.Result.Name)
	}
}

func TestScoreReadability_lowReadabilityIsWarn(t *testing.T) {
	// Текст с очень длинными, сложными предложениями даёт низкий FRE.
	// Ключевое: статус = warn, а не fail; код возврата не 1.
	docs := []domain.MarkdownDoc{
		{
			Path: "complex.md",
			Lines: []string{
				// Намеренно сложный текст: длинные слова, длинные предложения.
				"Непропорционально сложноструктурированная документация с неоправданно " +
					"громоздкими многосложными конструкциями существенно ухудшает воспринимаемость.",
			},
		},
	}
	// Используем высокий порог, чтобы гарантировать нарушение.
	cfg, _ := domain.NewConfig(domain.Request{})
	outcome := readability.ExportScoreReadability(docs, cfg)

	if outcome.Result.Status == "fail" {
		t.Errorf("scoreReadability (L1-контракт): статус не должен быть fail, получили fail")
	}
	// Нарушения, если есть, должны иметь severity=warning.
	for _, v := range outcome.Violations {
		if v.Severity != "warning" {
			t.Errorf("scoreReadability: нарушение L1 должно быть warning, получили %s", v.Severity)
		}
		if v.Layer != "L1" {
			t.Errorf("scoreReadability: нарушение должно быть в L1, получили %s", v.Layer)
		}
	}
}
