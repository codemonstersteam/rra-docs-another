# S3 — jtbd (L4 JTBD-присутствие)

Вход: `CLI rra-docs-another jtbd <path>`. Новых интеграций нет. Детерминированный
чеклист присутствия секций под каждого из четырёх потребителей.

## Дерево модулей

```
ParseArgs(args)                                -> Request
ProcessJTBD(req) [Deps: RepoStore]             -> (Report, error)
   | NewAuditTarget(req)                       -> AuditTarget
   | NewConfig(req)                            -> Config
   | store.ReadMarkdownDocs(target)            -> []MarkdownDoc
   | matchHeadings(docs, cfg)                  -> HeadingIndex
   | buildJTBDCard(index, spec) ×4             -> map[role]JTBDResult
   | buildReport(target, cmd, jtbdByRole)      -> Report
```

`buildJTBDCard` вызывается четырежды с разными `consumerSpec` (value-config):
`maintainer`, `consumer`, `manager`, `agent`. Четыре независимых результата — не
усредняются. Результаты собираются в `map[string]JTBDResult` с ключом-ролью.

## Псевдокод пайпа

```
ProcessJTBD(req) -> Result<Report, Error>:
    | NewAuditTarget(req)            -> AuditTarget              # path_not_found / read_error
    | NewConfig(req)                 -> Config                   # config_invalid
    | store.ReadMarkdownDocs(target) -> []MarkdownDoc            # read_error
    | matchHeadings(docs, cfg)       -> HeadingIndex
    | jtbdByRole := {
    |     "maintainer": buildJTBDCard(index, specMaintainer),
    |     "consumer":   buildJTBDCard(index, specConsumer),
    |     "manager":    buildJTBDCard(index, specManager),
    |     "agent":      buildJTBDCard(index, specAgent),
    |   }
    | buildReport(target, "jtbd", jtbdByRole) -> Report          # Report.JTBD = jtbdByRole
```

Четыре вызова `buildJTBDCard` — не цикл, а фиксированная развёртка по четырём
потребителям (детерминированно, читается за взгляд).

## Плюминг JTBD-карты — фиксировано (вариант b)

Локальный `buildReport` слайса принимает `jtbdByRole map[string]domain.JTBDResult`
**отдельным параметром**. `domain.ReportParts.JTBD` **не меняем** — пусть остаётся
`[]JTBDResult` для совместимости с S1/S2 (они это поле не трогают). Слайс
самодостаточен: своя сборка отчёта, без общего контейнера на четыре роли. Когда
S5/S7 повторят паттерн — каждый сделает свой `buildReport`, дубль по соглашению.

`layers.L4` подкомандой `jtbd` **не заполняется** — её соберёт S7 `assess` (там
послойная картина L1–L6 нужна для рендера). Здесь только `Report.JTBD = …` и
`target`/`command`/`schema_version`/`tool`.

## Контракты модулей

### matchHeadings
- **Сигнатура:** `matchHeadings(docs []MarkdownDoc, cfg Config) -> HeadingIndex`
- **Input (data):** `[]MarkdownDoc`. **Dependencies:** `Config` (на будущее —
  пороги; сами словари синонимов живут константами в `logic.go` слайса, чтобы
  `Config` оставался про числовые пороги; кастомизация словарей — поздний S3+).
- **Что делает:** нормализует H1–H3 и индексирует совпадения по словарю.
- **Консеквент:** `HeadingIndex` — карта «нормализованный заголовок → file:line».

### buildJTBDCard
- **Сигнатура:** `buildJTBDCard(index HeadingIndex, spec consumerSpec) -> JTBDResult`
- **Input (data):** `HeadingIndex`, `consumerSpec` (обязательные секции роли).
- **Что делает:** сверяет обязательные секции роли с индексом.
- **Консеквент:** все секции есть → `PASS`; часть некритичных пропущена →
  `PARTIAL`; хотя бы одна `critical` отсутствует → `FAIL`. `Gaps` — отсутствующие
  секции; `Score` 0–100.

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
