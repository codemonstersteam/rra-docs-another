# S4 — style (L2 стиль)

Вход: `CLI rra-docs-another style <path>`. Новая интеграция: `LinterRunner`
(subprocess Vale / markdownlint-cli2). Линтер сканирует директорию репо целиком —
без цикла по докам в голове.

## Дерево модулей

```
parseStyleArgs(args)                       -> Request
runStyle(req) [Deps: LinterRunner]         -> (Report, error)
   | NewAuditTarget(req)                   -> AuditTarget
   | NewConfig(req)                        -> Config            # профиль линтера
   | linter.Run(target)                    -> StyleFindings     # I/O: subprocess
   | aggregateFindings(findings)           -> LayerOutcome      # чистая логика L2
   | buildReport({Layers:[outcome]})       -> Report
```

## Псевдокод пайпа

```
runStyle(req) -> Result<Report, Error>:
    | NewAuditTarget(req)        -> AuditTarget
    | NewConfig(req)             -> Config
    | linter.Run(target)         -> StyleFindings
    | aggregateFindings(findings)-> LayerOutcome
    | buildReport({Layers:[outcome]}) -> Report
```

Профиль линтера передаётся в `LinterRunner` при его конструировании (value-config),
не как data-аргумент в голову.

## Контракты модулей

### linter.Run (I/O)
- **Сигнатура:** `Run(target AuditTarget) -> Result<StyleFindings, Error>`
- **Input (data):** AuditTarget. **Dependencies:** — (subprocess + профиль инкапсулированы).
- **Что делает:** запускает Vale/markdownlint на директории, парсит вывод.
- **Консеквент:** Success — `StyleFindings`. Failure — `ErrToolMissing` (бинаря
  нет в `PATH`), `ErrToolFailed` (линтер упал не из-за находок). Маппинг кода
  возврата линтера в доменную ошибку — единственное ветвление в I/O-модуле.

### aggregateFindings
- **Сигнатура:** `aggregateFindings(findings StyleFindings) -> LayerOutcome`
- **Input (data):** StyleFindings. **Dependencies:** —
- **Что делает:** маппит находки линтера в `[]Violation` (layer L2), считает статус.
- **Консеквент:** `Status=fail` при наличии blocker-правил; иначе `pass`/`warn`.

(`NewAuditTarget`, `NewConfig`, `buildReport` — см. S1.)

## Юнит-тесты

| Модуль | Happy | Ветки | Итого |
|---|---|---|---|
| aggregateFindings | 1 | есть blocker-находка → fail | 2 |

`linter.Run` — труба, не юнитится: success зеленит happy-сценарий, failure-ветки
(`tool_missing`, `tool_failed`) — сценарии отказа.

## Решения по дизайну

- Контракт `LinterRunner.Run` принимает `AuditTarget` (директорию), а не
  `MarkdownDoc` (как в наброске `PLAN.md`): линтеры сканируют дерево, это убирает
  цикл по докам из головы. Зафиксировано в `infrastructure.md` и `contracts-graph.md`.
- Установка Vale/markdownlint в образ раннера и Given-степы «линтеры
  недоступны»/«линтер падает» реализуются вместе с этим слайсом (сейчас `@wip`).

## Gherkin-mapping (`features/style.feature`)

| Сценарий | Then-шаг | Кто обеспечивает |
|---|---|---|
| опрятный репозиторий | код возврата 0 | egress `exitCode` |
| опрятный репозиторий | `layers.L2.status` = "pass" | `aggregateFindings` (Success) |
| линтер не установлен | код возврата 2 | egress ← `linter.Run` (`ErrToolMissing`) |
| линтер не установлен | `errors[]` `tool_missing` | `buildErrorReport` |
| линтер завершается с ошибкой | код возврата 2 | egress ← `linter.Run` (`ErrToolFailed`) |
| линтер завершается с ошибкой | `errors[]` `tool_failed` | `buildErrorReport` |

[x] Gherkin-mapping сверен
