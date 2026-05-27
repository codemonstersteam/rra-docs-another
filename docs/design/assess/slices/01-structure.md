# S1 — structure (L3 структурная полнота)

Вход: `CLI rra-docs-another structure <path>`. Новые интеграции: `RepoStore`,
`ReportSink` (+ общий egress). Этот слайс ставит I/O-фундамент.

## Дерево модулей

```
parseStructureArgs(args)              -> Request          # ингресс-адаптер: только парсинг
runStructure(req) [Deps: RepoStore]   -> (Report, error)  # голова (труба)
   | NewAuditTarget(req)              -> AuditTarget       # конструктор: валидация пути
   | NewConfig(req)                   -> Config            # конструктор: валидация --config
   | store.ReadStructure(target)      -> RepoStructure     # I/O: RepoStore
   | checkStructure(structure)        -> LayerOutcome      # чистая логика L3  [dep: Config]
   | buildReport(parts)               -> Report            # чистая логика (общий)  [dep: target,"structure"]
```

`checkStructure` агрегирует чистые под-проверки: `checkReadmePresent`,
`checkLinksResolve`, `checkDocDrift` (доки старше кода > N дней, N из Config).

## Псевдокод пайпа

```
runStructure(req) -> Result<Report, Error>:
    | NewAuditTarget(req)         -> AuditTarget
    | NewConfig(req)              -> Config
    | store.ReadStructure(target) -> RepoStructure
    | checkStructure(structure)   -> LayerOutcome        # [dep: Config]
    | buildReport({Layers:[outcome]}) -> Report
```

Запись и код возврата — общий egress (`infrastructure.md`). Ошибки I/O
(`ErrPathNotFound`, `ErrReadError`) пробрасываются без трансформации.

## Контракты модулей

### NewAuditTarget
- **Сигнатура:** `NewAuditTarget(req Request) -> Result<AuditTarget, Error>`
- **Input (data):** Request. **Dependencies:** —
- **Что делает:** валидирует путь репозитория.
- **Антецедент:** `req.Path` резолвится в существующую читаемую директорию.
- **Консеквент:** Success — `AuditTarget{root,commit}`. Failure — `ErrPathNotFound`, `ErrReadError`.

### NewConfig
- **Сигнатура:** `NewConfig(req Request) -> Result<Config, Error>`
- **Input (data):** Request. **Dependencies:** —
- **Антецедент:** `ConfigPath` пуст (дефолты) ИЛИ файл парсится по схеме.
- **Консеквент:** Success — `Config`. Failure — `ErrConfigInvalid`.

### checkReadmePresent / checkLinksResolve / checkDocDrift
- **Сигнатура:** `(structure RepoStructure) -> []Violation`
- **Input (data):** RepoStructure. **Dependencies:** `checkDocDrift` — `Config` (порог N).
- **Антецедент:** —. **Консеквент:** список нарушений L3 (severity blocker для отсутствующего README и битых ссылок; warning для дрейфа по возрасту).

### checkStructure
- **Сигнатура:** `checkStructure(structure RepoStructure) -> LayerOutcome`
- **Input (data):** RepoStructure. **Dependencies:** `Config`.
- **Что делает:** агрегирует под-проверки в `LayerOutcome` (L3).
- **Консеквент:** `Status=fail` при наличии blocker; иначе `pass`/`warn`. `Score` — доля пройденных проверок.

### buildReport (общий)
- **Сигнатура:** `buildReport(parts ReportParts) -> Report`
- **Input (data):** ReportParts. **Dependencies:** AuditTarget, command (`"structure"`).
- **Консеквент:** `Report` с `Layers.L3`, `Target`, `Command`, агрегированными `Violations`.

### store.ReadStructure (I/O)
- **Сигнатура:** `ReadStructure(target AuditTarget) -> Result<RepoStructure, Error>`
- **Input (data):** AuditTarget. **Dependencies:** — (ФС инкапсулирована).
- **Консеквент:** Success — `RepoStructure`. Failure — `ErrReadError`.

## Юнит-тесты (формула `1 happy + Σ ветки антецедента`)

| Модуль | Happy | Ветки | Итого |
|---|---|---|---|
| NewAuditTarget | 1 | путь не найден; нет прав | 3 |
| NewConfig | 1 | битый конфиг | 2 |
| checkReadmePresent | 1 | README отсутствует | 2 |
| checkLinksResolve | 1 | битая ссылка | 2 |
| checkDocDrift | 1 | доки устарели | 2 |
| checkStructure | 1 | есть blocker → fail | 2 |
| buildReport | 1 | — | 1 |

Голова `runStructure`, I/O `store.ReadStructure`, ингресс-адаптер, egress — **не**
юнитятся (трубы); зеленятся компонентными сценариями.

## Gherkin-mapping (`features/structure.feature`)

| Сценарий | Then-шаг | Кто обеспечивает |
|---|---|---|
| опрятный репозиторий проходит | код возврата 0 | egress `exitCode` (нет blocker/errors) |
| опрятный репозиторий проходит | `command` = "structure" | `buildReport` (dep command) |
| опрятный репозиторий проходит | `layers.L3.status` = "pass" | `checkStructure` (Success) |
| битый репозиторий | код возврата 1 | egress `exitCode` (blocker present) |
| битый репозиторий | `layers.L3.status` = "fail" | `checkStructure` (Failure-ветка) |
| путь не существует | код возврата 2 | egress (err≠nil) ← `NewAuditTarget` |
| путь не существует | `errors[]` `path_not_found` | `buildErrorReport` ← `ErrPathNotFound` |

[x] Gherkin-mapping сверен
