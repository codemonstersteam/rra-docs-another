# Backlog — rra-docs-another

Порядок — из [`PLAN.md`](./PLAN.md): гейт (контракт + Gherkin) → проектный пакет →
семь слайсов. Правило «один тикет = один slice = одна ветка = один PR».

---

## E0. Каркас (chore)

Go-модуль, раскладка, CLI-роутер, `version`, CI на PR. **Готово:** main зелёный.

## E1. Контракт + Gherkin (гейт program-design)

`docs/intent.md` (есть), `api-specification/cli.md`, `report.schema.json`, README
«Карта режимов отказа», godog-раннер + фикстуры `repo-good`/`repo-bad`, Gherkin на
подкоманды. **Готово:** smoke зелёный, контракт зафиксирован.

## E2. Проектный пакет (program-design, Шаги 1–12)

`docs/design/assess/`: `slices.md`, `messages.md`, карточки слайсов,
`infrastructure.md`, `contracts-graph.md`, `backlog.md` с хендофф-чеклистом.
**Готово:** дизайн-PR смержен (= аппрув).

## E3–E9. Слайсы (program-implementation)

| Тикет | Slice | Подкоманда | Слой | Новые I/O |
|---|---|---|---|---|
| S1 | `structure` | `structure` | L3 | RepoStore, ReportSink |
| S2 | `readability` | `readability` | L1 | — |
| S3 | `jtbd-presence` | `jtbd` | L4 | — |
| S4 | `style` | `style` | L2 | LinterRunner |
| S5 | `jtbd-fitness` | `fitness` | L5 | LLMClient |
| S6 | `drift` | `drift` | L6a | — |
| S7 | `assess` | `assess` | L1–L6 | — |
| S8 | `drift --semantic` | (флаг S6, поздний) | L6c | LLMClient |

LLM появляется только в базовом S5 и опциональном позднем S8. S6 детерминированный
(дрейф документации), работает на любой репе.

## E10. Эталонные фикстуры

`repo-good` / `repo-bad` + снэпшот-тесты. **Готово:** регрессии ловятся снэпшотами.

---

## Принципы работы с backlog

- Тикет не стартует без `intent` слайса и согласованного `contracts-graph.md`.
- TBD: main всегда зелёный, ветки живут часы–день.
- Не предполагать дисциплину; работает на произвольном репо.
- При противоречии в спецификации — **остановиться и сообщить**.
