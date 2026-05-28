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

## E1.1. Дотянуть обвязку компонент-тестов до полноты контракта

Раннер и smoke зелёные, но контракт `cli.md` + `report.schema.json` триггерится
не целиком. Цель — закрыть spec-уровневые пробелы единым PR. Узких фикстур
«под один слой» **не делать** — это юнит-уровень (skill `component-tests`,
секция «Граница со слоем юнитов»).

Scope:

- шаг `отчёт валидируется по api-specification/report.schema.json` + применить
  в каждом happy-сценарии;
- в happy-сценариях ассертить `command`, `schema_version`, `tool`, `target.path`;
- `git init && commit` в фикстурах (`Dockerfile.runtime`), чтобы покрыть
  `target.commit` в варианте `string`;
- сценарии под не покрытые `error.code`: `read_error` (фикстура с `chmod 000`),
  `config_invalid` (битый `--config`);
- ассертить `errors[].integration` рядом с `error.code` (контракт схемы);
- `help` / `--help` — код 0, usage в stdout;
- `--format md` для одной подкоманды (минимум: stdout — markdown, не JSON);
- `--out <файл>` против `--out -`;
- `assess --up-to L4` со `layers.L5.status="skipped"` / `L6.status="skipped"`
  (контрактная необходимость узкой фикстуры: репа, ломающая ранний слой и не
  ломающая поздний);
- вынести знание `--llm-*` из `runOnRepo` в явный степ;
- в smoke сценарии, дёргающие каждый зарегистрированный степ
  (требование skill `component-tests`, чек-лист хендоффа).

Out of scope:

- `tool_missing` / `tool_failed` — приезжают с S4 (style);
- `llm_*` — уже покрыты в `fitness.feature`;
- `--format md` для всех подкоманд (рендер — юнит-уровень).

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
