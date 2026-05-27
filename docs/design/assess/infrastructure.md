# infrastructure — assess

Технический корень. Бизнес-логики нет: собирает слайсы и поднимает CLI. Слайс —
**самодостаточный пакет** со строгим набором файлов (конвенция как в
`ubik/passkey-demo-api`); кросс-сквозная инфраструктура — в общих пакетах
`internal/{io,report,audit,cli}` (аналог их `internal/{db,clock,app}`).

## Конвенция слайса (файлы)

`internal/slice/<name>/` — один пакет, один внешний вход (CLI-подкоманда):

| Файл | Содержимое |
|---|---|
| `head.go` | **голова** `Process<Slice>(req, Deps) -> (Report, error)` — оркестратор-труба |
| `adapter.go` | ингресс-адаптер: парсинг аргументов → `Request` (их `handler.go`) |
| `logic.go` (+`_test.go`) | чистые логика-листья слоя |
| `domain.go` (+`_test.go`) | типы/конструкторы, специфичные слайсу (если есть) |
| `errors.go` | sentinel-ошибки слайса (если есть свои) |
| `register.go` | `Deps` + `Register(router, deps)` — слайс сам подключает свою подкоманду |

Голова именуется `Process<Slice>` и лежит в `head.go` — её видно сразу (как
`ProcessUsersMe`/`ProcessRegistrationStart` в эталоне).

## Голова слайса (`head.go`)

```
Process<Slice>(req Request, deps Deps) -> (Report, error):
    | NewAuditTarget(req)          -> AuditTarget      # audit (общий конструктор)
    | NewConfig(req)               -> Config
    | deps.Store.Read…(target)     -> данные            # io.RepoStore (общий)
    | <чистые листья слоя>         -> LayerOutcome / []JTBDResult
    | buildReport(parts, target, "<slice>") -> Report   # report (общий)
```

Голова — труба: каждый шаг зовёт один модуль, ошибки I/O пробрасываются без
трансформации. Запись отчёта и код возврата — общий egress (ниже), не в голове.

## Само-подключение (`register.go`)

Слайс сам регистрирует свою подкоманду — роутер не хардкодит каждый слайс
(как `mux.Get(path, …)` в эталоне):

```
Deps struct { Store *io.RepoStore; … }              # зависимости слайса, инжектит cli

Register(router *cli.Router, deps Deps):
    router.Handle("<slice>", func(args, stdout, stderr) int {
        | req, err := ParseArgs(args, stderr)        # adapter.go
        | if err != nil { … return 2 }
        | report, runErr := Process<Slice>(req, deps) # head.go
        | return report.Egress(report, runErr, req, deps.Sink, stdout)  # общий egress
    })
```

## Общий egress (`internal/report`)

Один на все слайсы — отчёт у тула машинно-единый (одна схема, один набор
`error.code`), поэтому egress общий, а не per-slice mapError как в HTTP-эталоне.

```
Egress(report, err, req, sink, stdout) -> int:
    | если err != nil: report = buildErrorReport(req, err)   # sentinel → Error{code}
    | sink.Write(report, req.Format, req.Out|stdout)         # io.ReportSink
    | return exitCode(report)                                # 0/1/2
```

`buildErrorReport` и `exitCode` — чистые функции (юнитятся). Правило `exitCode`:
`Errors` → 2; иначе blocker-`Violation` или JTBD `FAIL` → 1; иначе 0. Нарушения
`layer:"L1"` никогда не blocker → `readability` сам по себе кода 1 не даёт.

## Общая инфраструктура (общие пакеты)

| Пакет | Содержимое | Аналог в passkey |
|---|---|---|
| `internal/cli` | роутер подкоманд + сборка `Deps` слайсов (wiring) | `app/wire.go` + main |
| `internal/audit` | `AuditTarget`, `Config` + конструкторы `NewAuditTarget`/`NewConfig` | (доменные типы-владельцы) |
| `internal/report` | `Report`, `LayerResult`, `JTBDResult`, `Violation`, `Error` + egress | — |
| `internal/io` | автономные I/O-объекты (ниже) | `internal/db` |

Эти вещи используют все слайсы — отдельного «владельца-слайса» у них нет, поэтому
они общие (это не нарушение конвенции: у эталона общие `db`/`clock`/`app`).

## I/O-объекты (автономные, скрывают зависимость) — `internal/io`

| Объект | Скрывает | Метод (контракт) | Слайсы |
|---|---|---|---|
| `RepoStore` | ФС / git | `ReadMarkdownDocs(AuditTarget) -> ([]MarkdownDoc, error)`; `ReadStructure(AuditTarget) -> (RepoStructure, error)` | S1,S2,S3,S6,S7 |
| `LinterRunner` | subprocess Vale / markdownlint | `Run(AuditTarget) -> (StyleFindings, error)` — сканирует директорию (без цикла в голове) | S4,S7 |
| `LLMClient` | LLM (anthropic / openai-совм.) | `Simulate(JTBDPromptSet) -> ([]LLMVerdict, error)` — фан-аут 4 ролей внутри объекта; `Judge(ClaimPrompt) -> (Verdict, error)` (S8) | S5,S7,S8 |
| `ReportSink` | stdout / файл | `Write(Report, format, out) -> error` | все (через egress) |

Правила: в `Dependencies:`/`Deps` — только эти объекты, не сырые `*os`/`*exec`/`*http`.
Каждый метод — труба (одно сообщение → внешний вызов → результат/доменная ошибка);
единственное ветвление — маппинг кода внешней системы в `Err*`. I/O юнитами не
покрываются (success → happy-сценарий, failure → сценарий отказа).

Отказы (воспроизводимы в component-tests): RepoStore — `ErrPathNotFound`/
`ErrReadError`; LinterRunner — `ErrToolMissing`/`ErrToolFailed`; LLMClient —
`ErrLLMRateLimited`/`ErrLLMUnavailable`/`ErrLLMBudgetExceeded`.

## Конфигурация и секреты

- LLM-ключ только из env (`ANTHROPIC_API_KEY`/`OPENAI_API_KEY`), не из флага.
- `--config` валидируется `NewConfig`; отсутствие → встроенные дефолты.

## Раскладка кода

```
cmd/rra-docs-another/main.go     # тонкая точка входа
internal/cli/                    # роутер подкоманд + сборка Deps слайсов
internal/audit/                  # AuditTarget, Config + конструкторы
internal/report/                 # Report-типы + egress (buildErrorReport, exitCode)
internal/io/                     # RepoStore, LinterRunner, LLMClient, ReportSink
internal/slice/<name>/           # head.go · adapter.go · logic.go · domain.go ·
                                 #   errors.go · register.go  (самодостаточный пакет)
```
