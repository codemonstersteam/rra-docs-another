// Package readability реализует слайс S2 — L1 читаемость.
// Чистые функции: нет I/O, нет глобального состояния.
// Особое правило L1: статус ∈ {pass, warn}, никогда fail; код возврата 1 не возникает.
package readability

import (
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// ── текстовые примитивы ───────────────────────────────────────────────────────

// extractText извлекает читаемый текст из MarkdownDoc: удаляет код-блоки (```)
// и инлайн-код, снимает маркеры заголовков.
func extractText(doc domain.MarkdownDoc) string {
	var sb strings.Builder
	inFence := false
	for _, line := range doc.Lines {
		stripped := strings.TrimSpace(line)
		if strings.HasPrefix(stripped, "```") || strings.HasPrefix(stripped, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		// Снимаем маркер заголовка (#... текст).
		if strings.HasPrefix(stripped, "#") {
			i := 0
			for i < len(stripped) && stripped[i] == '#' {
				i++
			}
			stripped = strings.TrimSpace(stripped[i:])
		}
		// Убираем инлайн-код (`...`).
		stripped = removeInlineCode(stripped)
		sb.WriteString(stripped)
		sb.WriteString(" ")
	}
	return sb.String()
}

func removeInlineCode(s string) string {
	var sb strings.Builder
	inCode := false
	for _, r := range s {
		if r == '`' {
			inCode = !inCode
			continue
		}
		if !inCode {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// countWords считает пробельно-разделённые токены, содержащие хотя бы одну букву
// или цифру.
func countWords(text string) int {
	count := 0
	for _, w := range strings.Fields(text) {
		for _, r := range w {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				count++
				break
			}
		}
	}
	return count
}

// countSentences считает завершённые предложения (. ! ?).
// Минимум 1 (защита от деления на ноль).
func countSentences(text string) int {
	count := 0
	for _, r := range text {
		if r == '.' || r == '!' || r == '?' {
			count++
		}
	}
	if count == 0 {
		return 1
	}
	return count
}

// ── слоги ─────────────────────────────────────────────────────────────────────

// countSyllablesEnWord считает слоги в одном английском слове эвристикой:
// группы гласных (a e i o u y) = слоги; тихое -e в конце не считается.
func countSyllablesEnWord(word string) int {
	word = strings.ToLower(word)
	const vowels = "aeiouy"
	count := 0
	prev := false
	for _, r := range word {
		v := strings.ContainsRune(vowels, r)
		if v && !prev {
			count++
		}
		prev = v
	}
	// Тихое e в конце: слово длиннее двух букв, предпоследняя — согласная.
	if len(word) > 2 && strings.HasSuffix(word, "e") {
		penultimate := rune(word[len(word)-2])
		if !strings.ContainsRune(vowels, penultimate) && count > 1 {
			count--
		}
	}
	if count < 1 {
		count = 1
	}
	return count
}

// countSyllablesEnText считает суммарное количество слогов в английском тексте.
func countSyllablesEnText(text string) int {
	total := 0
	for _, w := range strings.Fields(text) {
		clean := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) {
				return r
			}
			return -1
		}, w)
		if clean != "" {
			total += countSyllablesEnWord(clean)
		}
	}
	return total
}

// countSyllablesRu считает слоги в русском тексте как количество гласных.
// Русские гласные: а е ё и о у ы э ю я (10 букв, й — согласный).
func countSyllablesRu(text string) int {
	const ruVowels = "аеёиоуыэюяАЕЁИОУЫЭЮЯ"
	count := 0
	for _, r := range text {
		if strings.ContainsRune(ruVowels, r) {
			count++
		}
	}
	return count
}

// ── язык ──────────────────────────────────────────────────────────────────────

// cyrillicRatio возвращает долю кириллических букв среди всех букв в тексте дока.
func cyrillicRatio(doc domain.MarkdownDoc) float64 {
	var total, cyrillic int
	for _, line := range doc.Lines {
		for _, r := range line {
			if unicode.IsLetter(r) {
				total++
				// Кириллический блок Unicode: U+0400–U+04FF.
				if r >= 0x0400 && r <= 0x04FF {
					cyrillic++
				}
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(cyrillic) / float64(total)
}

// ── формулы ───────────────────────────────────────────────────────────────────

// pickFormula выбирает формулу читаемости по языку документа:
// доля кириллицы ≥ 30% → obornevaRus, иначе → fleschKincaid.
// Для пустого текста → fleschKincaid (без паники).
func pickFormula(doc domain.MarkdownDoc) func(domain.MarkdownDoc) float64 {
	if cyrillicRatio(doc) >= 0.30 {
		return obornevaRus
	}
	return fleschKincaid
}

// fleschKincaid вычисляет Flesch Reading Ease для английского текста.
// FRE = 206.835 − 1.015·ASL − 84.6·ASW.
// Для пустого текста возвращает нейтральное значение 70 (без паники).
func fleschKincaid(doc domain.MarkdownDoc) float64 {
	text := extractText(doc)
	words := countWords(text)
	if words == 0 {
		return 70 // нейтральное значение для пустого документа
	}
	sentences := countSentences(text)
	syllables := countSyllablesEnText(text)
	asl := float64(words) / float64(sentences)
	asw := float64(syllables) / float64(words)
	score := 206.835 - 1.015*asl - 84.6*asw
	return clamp(score, 0, 100)
}

// obornevaRus вычисляет адаптированный Flesch Reading Ease для русского текста
// (формула Оборневой): FRE = 206.836 − 1.52·ASL − 65.14·ASW.
// Слоги ≈ гласные буквы. Для пустого текста — нейтральное значение 70.
func obornevaRus(doc domain.MarkdownDoc) float64 {
	text := extractText(doc)
	words := countWords(text)
	if words == 0 {
		return 70 // нейтральное значение для пустого документа
	}
	sentences := countSentences(text)
	syllables := countSyllablesRu(text)
	asl := float64(words) / float64(sentences)
	asw := float64(syllables) / float64(words)
	score := 206.836 - 1.52*asl - 65.14*asw
	return clamp(score, 0, 100)
}

func clamp(v, lo, hi float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ── агрегация ─────────────────────────────────────────────────────────────────

// scoreReadability вычисляет L1-исход для набора документов.
// Статус ∈ {pass, warn} — никогда fail (L1-контракт).
// Violations — только severity=warning.
func scoreReadability(docs []domain.MarkdownDoc, cfg domain.Config) domain.LayerOutcome {
	if len(docs) == 0 {
		score := 100
		return domain.LayerOutcome{
			Result: domain.LayerResult{
				Name:    "readability",
				Status:  "pass",
				Score:   &score,
				Summary: "нет документов для проверки читаемости",
			},
		}
	}

	threshold := float64(cfg.ReadabilityMin())
	var total float64
	var violations []domain.Violation

	for _, doc := range docs {
		formula := pickFormula(doc)
		docScore := formula(doc)
		total += docScore

		if docScore < threshold {
			violations = append(violations, domain.Violation{
				Code:     "low_readability",
				Layer:    "L1",
				Severity: "warning",
				File:     doc.Path,
				Message: fmt.Sprintf(
					"читаемость %.0f ниже порога %d (Flesch Reading Ease)",
					docScore, cfg.ReadabilityMin(),
				),
			})
		}
	}

	avg := total / float64(len(docs))
	scoreInt := int(math.Round(avg))
	scoreInt = int(clamp(float64(scoreInt), 0, 100))

	status := "pass"
	if len(violations) > 0 {
		status = "warn"
	}

	return domain.LayerOutcome{
		Result: domain.LayerResult{
			Name:    "readability",
			Status:  status,
			Score:   &scoreInt,
			Summary: buildReadabilitySummary(status, avg, len(violations)),
		},
		Violations: violations,
	}
}

func buildReadabilitySummary(status string, avg float64, warnings int) string {
	switch status {
	case "pass":
		return fmt.Sprintf("читаемость в норме (средний FRE %.0f)", avg)
	case "warn":
		return fmt.Sprintf("низкая читаемость: %d предупреждений (средний FRE %.0f)", warnings, avg)
	default:
		return ""
	}
}
