# План разработки rra-docs-another на Go — маленькими шагами

Проектирование и реализация идут строго по скиллам из `skills/`:
`program-design` (opus, проектный пакет) → `component-tests` (Gherkin как
исполняемая спецификация) → `program-implementation` (sonnet, TBD по тикетам).

Принцип нарезки: **один внешний вход = один slice = один тикет = одна ветка =
один PR**. Внешний вход — CLI-подкоманда. Семь подкоманд → семь слайсов, плюс
поздний тикет S8 (флаг семантического тира L6, не новая подкоманда).

---

## Жёсткий гейт program-design (Шаг 0)

Проектирование не стартует без зафиксированного контракта, карты режимов отказа в
README и Gherkin-сценариев. У CLI роль контракта играют:

1. `api-specification/cli.md` — команды, флаги, коды возврата.
2. `api-specification/report.schema.json` — схема машинного отчёта.

Поэтому фазы 1–3 идут **до** проектирования (фаза 4) и реализации (фаза 5).

## Фаза 0 — каркас (chore, 1 PR)

- `go mod init github.com/codemonstersteam/rra-docs-another`, Go 1.23+.
- Раскладка: `cmd/rra-docs-another/`, `internal/slice/<name>/`, `internal/io/`,
  `internal/domain/`.
- CLI-роутер (stdlib `flag` + subcommand switch).
- `... version`, `go test ./...` зелёный. CI на PR: `vet`, `test`, `gofmt`.

**Готово:** `version` печатается, CI зелёный.

## Фаза 1 — intent (Шаг 1)

`docs/intent.md` уже есть: «оценить качество документации произвольного репо для
четырёх JTBD-потребителей».

## Фаза 2 — контракт (гейт)

Ветка `feat/contract`.

- `api-specification/cli.md` — семь подкоманд, флаги, коды возврата `0/1/2`.
- `api-specification/report.schema.json` — четыре JTBD-секции
  (`PASS/FAIL/PARTIAL` + score + пробелы), список нарушений с `file:line`.
- README раздел **«Карта режимов отказа»**:

  | Интеграция | error.code | Код | Действие пользователя |
  |---|---|---|---|
  | RepoStore (ФС) | `path_not_found` / `read_error` | 2 | проверить путь/права |
  | LinterRunner | `tool_missing` / `tool_failed` | 2 | установить Vale/markdownlint |
  | LLMClient | `llm_rate_limited` / `llm_unavailable` / `llm_budget_exceeded` | 2 | ретрай / позже / поднять бюджет |

**Готово:** контракт зафиксирован, карта отказов покрывает интеграции.

## Фаза 3 — компонентные тесты (component-tests)

godog + фикстуры `testdata/repo-good` (любой опрятный репо) / `testdata/repo-bad`
+ базовые степы. LLM-сценарии бьют в **стаб LLM-эндпоинта**. На каждую подкоманду
— happy + сценарий на режим отказа её интеграций.

**Готово:** smoke зелёный, `.feature` покрывают подкоманды.

## Фаза 4 — проектный пакет (program-design, Шаги 1–12)

`docs/design/assess/`: `slices.md`, `messages.md` (`AuditTarget`, `MarkdownDoc`,
`ReadabilityScore`, `StyleFindings`, `StructureReport`, `JTBDCard`, `JTBDResult`,
`Claim`, `DriftFinding`, `Report`, `Violation`, `Error`), карточки слайсов
(дерево модулей + контракты + `## Gherkin-mapping` + таблица юнит-тестов),
`infrastructure.md`, `contracts-graph.md`, `backlog.md` с хендофф-чеклистом.

### I/O-объекты (Шаг 6)

| Объект | Скрывает | Контракт |
|---|---|---|
| `RepoStore` | ФС/git | `ReadMarkdownDocs(AuditTarget) -> ([]MarkdownDoc, error)`, `ReadStructure(AuditTarget) -> (StructureReport, error)` |
| `LinterRunner` | subprocess | `Run(MarkdownDoc) -> (StyleFindings, error)` |
| `LLMClient` | Anthropic API | `Simulate(JTBDPrompt) -> (JTBDResult, error)` |
| `ReportSink` | stdout/файл | `Write(Report) -> error` |

### Семь слайсов (порядок = дешёвое-и-ценное первым)

| # | Slice | Подкоманда | Слой | Новые интеграции | Логика-листья |
|---|---|---|---|---|---|
| S1 | `structure` | `structure` | L3 | RepoStore, ReportSink | `NewAuditTarget`, `checkStructure` |
| S2 | `readability` | `readability` | L1 | — | `fleschKincaid`, `obornevaRus`, `scoreReadability` |
| S3 | `jtbd-presence` | `jtbd` | L4 | — | `matchHeadings`, `buildJTBDCard` (×4) |
| S4 | `style` | `style` | L2 | LinterRunner | `aggregateFindings` |
| S5 | `jtbd-fitness` | `fitness` | L5 | LLMClient | `buildJTBDPrompt`, `scoreFitness` |
| S6 | `drift` | `drift` | L6a | — | `extractClaims`, `verifyAgainstRepo` |
| S7 | `assess` | `assess` | L1–L6 | — | `shortCircuit`, `mergeJTBD` |
| S8 | `drift --semantic` | расширение S6 (поздний) | L6c | LLMClient | `buildClaimPrompt`, `judgePair` |

S1 первым: ставит `RepoStore` + `ReportSink` + схему отчёта на простейшем слое.
S2–S3 — чистая логика, ноль интеграций. S4 добавляет `LinterRunner`. **S5 —
единственный базовый LLM-слайс.** S6 детерминированный (дрейф по извлечённым
утверждениям), работает на любой репе. S7 собирает пайплайн из уже
протестированных листьев (не вызывает головы других слайсов), реализует
short-circuit «не звать LLM, если L4 упал» и правило «четыре score, не
усредняем». S8 (поздний, опциональный) — семантический LLM-тир L6 за флагом.

### Пример дерева модулей слайса S2 `readability` (Шаг 3)

```
parseReadabilityArgs(args)              -> Request            # адаптер: только парсинг
runReadability(req) -> Result<Report, Error>:                # голова (труба)
   | NewAuditTarget(req)                -> AuditTarget        # конструктор: валидация пути
   | store.ReadMarkdownDocs(target)     -> []MarkdownDoc      # I/O: RepoStore
   | scoreReadability(docs)             -> ReadabilityScore   # чистая логика (FK + Оборнева)
   | buildReport(score)                 -> Report             # чистая логика
   | sink.Write(report)                 -> ()                 # I/O: ReportSink
```

Юнит-тесты: `NewAuditTarget` (happy + ветки), `fleschKincaid` (happy + пустой
текст), `obornevaRus` (happy), `scoreReadability` (happy). Голова, адаптер, I/O —
не юнитятся, их зеленит Gherkin.

## Фаза 5 — реализация по тикетам (program-implementation, TBD)

S1→S7 (S8 позже), восходящий порядок, никаких моков, каждый тикет — отдельная
ветка/PR, `go test` + компонентные тесты зелёные, pre-push grep-самопроверка.

**Готово по продукту:** `assess` на `repo-good` — четыре PASS; на `repo-bad` —
конкретные пробелы с `file:line` ещё до запуска LLM.

---

## Карта шагов на скиллы

| Фаза | Что делаем | Скилл / шаг |
|---|---|---|
| 0 | каркас, CLI-роутер, CI | — (chore) |
| 1 | `intent.md` | program-design Шаг 1 |
| 2 | CLI-контракт + JSON Schema + карта отказов | program-design Шаг 0 (гейт), component-tests Шаг 0 |
| 3 | godog-раннер + Gherkin | component-tests Шаги 0–5 |
| 4 | проектный пакет `docs/design/assess/` | program-design Шаги 1–12 |
| 5 | реализация S1→S7 по тикетам | program-implementation Шаги 0–9 |

## Принципы

- Дешёвое-первым: L1/L3/L4 и ядро L6 (ноль интеграций) → L2 (subprocess) → L5 и
  L6c (LLM). LLM зовётся только там, где дёшево нельзя.
- Не предполагать дисциплину: работает на произвольном репо.
- Не усреднять JTBD: четыре независимых score.
- RRA правдив сам себе: I/O изолирован в объектах, логика — чистые функции.
