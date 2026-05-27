# S3 — jtbd (L4 JTBD-присутствие)

Вход: `CLI rra-docs-another jtbd <path>`. Новых интеграций нет. Детерминированный
чеклист присутствия секций под каждого из четырёх потребителей.

## Дерево модулей

```
parseJtbdArgs(args)                   -> Request
runJtbd(req) [Deps: RepoStore]        -> (Report, error)
   | NewAuditTarget(req)              -> AuditTarget
   | NewConfig(req)                   -> Config              # словари заголовков (ru+en)
   | store.ReadMarkdownDocs(target)   -> []MarkdownDoc
   | matchHeadings(docs)              -> HeadingIndex        # [dep: Config]
   | buildJTBDCard(index)             -> JTBDResult ×4       # [dep: consumerSpec]
   | buildReport({JTBD: results})     -> Report
```

`buildJTBDCard` вызывается четырежды с разным `consumerSpec` (value-config):
`maintainer`, `consumer`, `manager`, `agent`. Четыре независимых результата — не
усредняются.

## Псевдокод пайпа

```
runJtbd(req) -> Result<Report, Error>:
    | NewAuditTarget(req)            -> AuditTarget
    | NewConfig(req)                 -> Config
    | store.ReadMarkdownDocs(target) -> []MarkdownDoc
    | matchHeadings(docs)            -> HeadingIndex          # [dep: Config]
    | buildJTBDCard(index)           -> JTBDResult  # maintainer  [dep: spec]
    | buildJTBDCard(index)           -> JTBDResult  # consumer
    | buildJTBDCard(index)           -> JTBDResult  # manager
    | buildJTBDCard(index)           -> JTBDResult  # agent
    | buildReport({JTBD:[…4]})       -> Report
```

Четыре вызова `buildJTBDCard` — не цикл, а фиксированная развёртка по четырём
потребителям (детерминированно, читается за взгляд).

## Контракты модулей

### matchHeadings
- **Сигнатура:** `matchHeadings(docs []MarkdownDoc) -> HeadingIndex`
- **Input (data):** []MarkdownDoc. **Dependencies:** `Config` (словари синонимов).
- **Что делает:** нормализует H1–H3 и индексирует совпадения по словарю.
- **Консеквент:** `HeadingIndex` — карта «нормализованный заголовок → file:line».

### buildJTBDCard
- **Сигнатура:** `buildJTBDCard(index HeadingIndex) -> JTBDResult`
- **Input (data):** HeadingIndex. **Dependencies:** `consumerSpec` (обязательные секции роли).
- **Что делает:** сверяет обязательные секции роли с индексом.
- **Консеквент:** все секции есть → `PASS`; часть → `PARTIAL`; критичные
  отсутствуют → `FAIL`. `Gaps` — отсутствующие секции; `Score` 0–100.

(`NewAuditTarget`, `NewConfig`, `buildReport`, `store.ReadMarkdownDocs` — см. S1.)

## Юнит-тесты

| Модуль | Happy | Ветки | Итого |
|---|---|---|---|
| matchHeadings | 1 | нет заголовков | 2 |
| buildJTBDCard | 1 | часть секций → PARTIAL; критичных нет → FAIL | 3 |

## Gherkin-mapping (`features/jtbd.feature`)

| Сценарий | Then-шаг | Кто обеспечивает |
|---|---|---|
| опрятный — все четыре PASS | `jtbd.maintainer.status` = "PASS" | `buildJTBDCard` (maintainer, Success) |
| опрятный — все четыре PASS | `jtbd.consumer.status` = "PASS" | `buildJTBDCard` (consumer) |
| опрятный — все четыре PASS | `jtbd.manager.status` = "PASS" | `buildJTBDCard` (manager) |
| опрятный — все четыре PASS | `jtbd.agent.status` = "PASS" | `buildJTBDCard` (agent) |
| опрятный — все четыре PASS | код возврата 0 | egress `exitCode` |
| битый — есть проваленный JTBD | код возврата 1 | egress `exitCode` (FAIL present) |
| битый — есть проваленный JTBD | `jtbd.agent.status` = "FAIL" | `buildJTBDCard` (agent, Failure-ветка) |
| путь не существует | код возврата 2 | egress ← `NewAuditTarget` |
| путь не существует | `errors[]` `path_not_found` | `buildErrorReport` |

[x] Gherkin-mapping сверен
