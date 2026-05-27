# infrastructure — assess

Технический корень программы. Бизнес-логики нет ни строки: собирает слайсы из
готовых частей и поднимает CLI. Тип входов у всех слайсов — `CLI`, поэтому
инфраструктура = CLI-роутер + общий egress + конструкторы I/O-объектов.

## CLI-роутер (`internal/cli`, каркас E0)

`Run(args, stdout, stderr) -> int` диспетчеризует подкоманду в её ингресс-адаптер.
Для каждой подкоманды:

```
Run:
  | parse<Slice>Args(args)         -> Request           # ингресс-адаптер слайса
  | собрать Deps слайса (I/O-объекты)
  | run<Slice>(req, deps)          -> (Report, error)   # головной модуль слайса
  | egress(report, err, req, sink) -> int               # общий выход (см. ниже)
```

Роутер не содержит оркестрации между слайсами — слайсы независимы.

## Общий egress (единая точка «результат → ответ»)

Один на все слайсы. Заменяет инлайн `sink.Write` из эскиза скилла: и успех, и
отказ форматируются здесь, в одном месте.

```
egress(report Report, err error, req Request, sink ReportSink) -> int:
  | если err != nil:
  |     report = buildErrorReport(req, err)   # маппинг доменной ошибки → Error → Report{Errors:[…]}
  | sink.Write(report, req.Format, req.Out)   # I/O: ReportSink
  | return exitCode(report)                   # чистая логика
```

- `buildErrorReport(req, err) -> Report` — чистая логика: разворачивает sentinel
  (`errors.Is`) в `Error{Code, Integration}` по таблице из `messages.md`.
- `exitCode(report Report) -> int` — чистая логика:
  - есть `Errors` → `2`;
  - иначе есть `Violation{severity:blocker}` ИЛИ `JTBDResult{status:FAIL}` → `1`;
  - иначе `0`.
  - Исключение L1: нарушения `layer:"L1"` никогда не `blocker` (по контракту),
    поэтому `readability` сам по себе кода `1` не даёт.

`buildErrorReport` и `exitCode` — чистые функции, юнитятся по формуле. `egress`
как таковой — труба (I/O `sink.Write`), не юнитится; покрыт компонентными
сценариями (коды возврата + `errors[]`).

## I/O-объекты (автономные, скрывают зависимость)

| Объект | Скрывает | Метод (контракт) | Появляется в |
|---|---|---|---|
| `RepoStore` | ФС / git | `ReadMarkdownDocs(AuditTarget) -> ([]MarkdownDoc, error)`; `ReadStructure(AuditTarget) -> (RepoStructure, error)` | S1, S2, S3, S6, S7 |
| `LinterRunner` | subprocess Vale / markdownlint | `Run(AuditTarget) -> (StyleFindings, error)` — сканирует директорию репо целиком (без цикла по докам в голове) | S4, S7 |
| `LLMClient` | LLM (anthropic native / openai-совм.) | `Simulate(JTBDPromptSet) -> ([]LLMVerdict, error)` — фан-аут 4 ролей инкапсулирован в объекте (без цикла в голове); `Judge(ClaimPrompt) -> (Verdict, error)` (S8) | S5, S7, S8 |
| `ReportSink` | stdout / файл | `Write(Report, format, out) -> error` | все (через egress) |

Правила (Шаг 6 скилла):
- В `Dependencies:` контрактов логики и в `Deps` голов — **только** эти объекты,
  никогда сырые `*os.File`, `*exec.Cmd`, `*http.Client`.
- Каждый I/O-метод — труба: одно доменное сообщение → внешний вызов → результат
  или доменная ошибка. Единственное ветвление — маппинг кодов внешней системы в
  доменные ошибки (`SQLITE`-аналог: `exit≠0 линтера → ErrToolFailed`,
  `HTTP 429 → ErrLLMRateLimited`).
- I/O-модули юнитами не покрываются: success-ветка зеленит happy-сценарий,
  failure-ветки — сценарии отказа из `component-tests/features/`.

### Конкретика отказов I/O (для воспроизводимости в component-tests)

- `RepoStore`: `ErrPathNotFound` (нет директории), `ErrReadError` (нет прав).
- `LinterRunner`: `ErrToolMissing` (бинарь не в `PATH`), `ErrToolFailed`
  (линтер вернул ненулевой код не из-за находок, а из-за своей ошибки).
- `LLMClient`: `ErrLLMRateLimited` (429), `ErrLLMUnavailable` (нет ключа / 5xx /
  сеть), `ErrLLMBudgetExceeded` (учёт токенов превысил бюджет роли).

## Конфигурация и секреты

- LLM-ключ только из env (`ANTHROPIC_API_KEY` / `OPENAI_API_KEY`), не из флага.
- `--config` валидируется конструктором `NewConfig`; отсутствие → встроенные
  дефолты (словари заголовков, профиль линтеров, пороги читаем…).

## Раскладка кода

```
cmd/rra-docs-another/main.go     # тонкая точка входа (E0)
internal/cli/                    # роутер + общий egress (buildErrorReport, exitCode)
internal/domain/                 # messages.md: типы + конструкторы (NewAuditTarget, …)
internal/io/                     # RepoStore, LinterRunner, LLMClient, ReportSink
internal/slice/<name>/           # ингресс-адаптер + голова + логика-листья слайса
```
