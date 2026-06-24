# Egress — общий выход CLI (рефактор по уроку D1)

Кросс-срезовый узел (`internal/report` + `internal/io.ReportSink`), один на все 7
подкоманд. Своя design-карта — требование Conformance-gate (`program-design`):
сквозная инфраструктура не «падает между слайсами», на неё запускается сверка.

Эта карта **уточняет** раздел «Общий egress» в [`infrastructure.md`](infrastructure.md).

## Intent

Голова слайса вернула `(Report, error)` — egress превращает это в записанный
отчёт + код возврата, уважая `req.Out` (куда) и `req.Format` (как).

## Дефект D1 (что чиним) и его класс

`internal/cli/cli.go:191` зовёт `sink.WriteTo(report, req.Format, stdout)` — всегда
stdout, `req.Out` игнорируется → флаг `--out` no-op. Корень — расхождение с
дизайном + латентные смеллы дизайна:

1. **Side-инъекция `stdout io.Writer`** через `Run → runXCmd → egress` — внешний
   выбор «куда писать» проносится мимо `Request`. Нарушает «единый запрос» (Шаг 3).
2. **Test-only второй метод I/O** `WriteTo(io.Writer)` рядом с `Write(out)` —
   стал боевым путём в обход `req.Out`. Нарушает «запрет test-only метода» (Шаг 6).
3. **Рендер `format → content` в I/O-трубе** (switch в `Write` и `WriteTo`,
   продублирован) — должен быть чистой логикой (Шаг 3 п.2).
4. **`Write(report, format, out)` — 3 data-аргумента** — нарушает «один вход» (Шаг 3).

## Каталог сообщений (дополнение к `messages.md`)

- `Destination` — куда писать. Неэкспортируемые поля; создаётся `resolveDestination`.
  `Kind ∈ {Stdout, File}`, `Path string` (для File).
- `ReportOutput` — готовый к записи отчёт. Неэкспортируемые поля `content string`,
  `dest Destination`. Создаётся **только** конструктором `NewReportOutput`.

## Дерево модулей

```
internal/report/
  egress.go    Egress(report, err, req, sink) -> int        # оркестратор-труба
               buildErrorReport(req, err) -> Report          # чистая (есть)
               exitCode(report) -> int                       # чистая (есть)
  output.go    NewReportOutput(report, req) -> (ReportOutput, error)  # конструктор-узел
  render.go    renderReport(report, format) -> (string, error)        # чистый хелпер
  destination.go resolveDestination(req) -> (Destination, error)      # чистый хелпер

internal/io/
  reportsink.go ReportSink.Write(out ReportOutput) -> error  # I/O-труба, 1 вход
                                                             # (WriteTo — УДАЛИТЬ)
```

## Контракты модулей

### NewReportOutput (конструктор-узел)
- **Сигнатура:** `NewReportOutput(report Report, req Request) -> (ReportOutput, error)`
- **Input (data):** `Report` + `Request` — санкционированная сборка в конструкторе.
- **Dependencies:** —
- **Что делает:** рендерит отчёт по `req.Format` и резолвит `req.Out` в `Destination`,
  собирая единый `ReportOutput`.
- **Антецедент:** `req.Format ∈ {"", "json", "md"}`.
- **Консеквент:** Success — `ReportOutput{content, dest}` готов к записи.
  Failure — `ErrUnknownFormat` (формат вне множества).

### renderReport (чистый хелпер)
- **Сигнатура:** `renderReport(report Report, format string) -> (string, error)`
- **Что делает:** сериализует отчёт (`json`/`md`).
- **Антецедент:** `format ∈ {"", "json", "md"}`.
- **Консеквент:** Success — строка контента. Failure — `ErrUnknownFormat`.

### resolveDestination (чистый хелпер)
- **Сигнатура:** `resolveDestination(req Request) -> (Destination, error)`
- **Что делает:** `req.Out == "" | "-"` → `Stdout`; иначе `File{Path: req.Out}`.
- **Консеквент:** Success — `Destination`. Failure — нет (резолв тотален).

### ReportSink.Write (I/O-труба)
- **Сигнатура:** `Write(out ReportOutput) -> error`
- **Input (data):** один `ReportOutput`.
- **Dependencies:** — (ФС/`os.Stdout` инкапсулированы в объекте).
- **Что делает:** пишет `out.content` в `out.dest` (`Stdout` → `os.Stdout`;
  `File` → `os.WriteFile`). Без ветвлений по данным, кроме маппинга ошибки ФС.
- **Консеквент:** Success — байты записаны. Failure — `ErrReportWrite` (ошибка ФС).

## Псевдокод пайпа (egress — труба, без обходных путей)

```
Egress(report, runErr, req, sink) -> int:
    | если runErr != nil: report = buildErrorReport(req, runErr)   # sentinel → Error{code}
    | NewReportOutput(report, req)        -> ReportOutput           # ErrUnknownFormat
    | sink.Write(out)                     -> error                  # [I/O] ErrReportWrite
    | при ошибке любого шага: вернуть код 2
    | exitCode(report)                                              # 0/1/2
```

Параметр `stdout io.Writer` **удалён** из `Egress`/`Run`/`runXCmd`/`main`: «куда
писать» живёт в `req.Out`, `Request` — единственный носитель внешнего входа.

## Граф вызовов (сверка)

```
runXCmd (роутер)
   | req: Request (из adapter.go), report: Report (из головы)
   v
Egress
   |-- buildErrorReport(req, err)   -> Report          # только при runErr
   |-- NewReportOutput(report, req) -> ReportOutput     # рендер + назначение
   |       renderReport(report, format) -> string
   |       resolveDestination(req)      -> Destination
   |-- sink.Write(out: ReportOutput)-> error            # I/O
   |-- exitCode(report)             -> int
```

- [x] `req.Out`/`req.Format` (консеквент `ParseArgs`) потребляются `NewReportOutput`
  (антецедент). Производится — значит потреблено (исправление D1, Шаг 9 п.3).
- [x] Каждый узел — один data-вход (`NewReportOutput` — конструктор, сборка
  санкционирована; `sink.Write` — один `ReportOutput`).
- [x] Сырых `*os`/`io.Writer` в `Dependencies:`/`Deps` нет.

## Тесты (Шаг 8.1)

Юнит по формуле `1 happy + Σ ветки`:

| Модуль | Happy | Ветки | Итого |
|---|---|---|---|
| `renderReport` | json | md; неизвестный формат → `ErrUnknownFormat` | 3 |
| `resolveDestination` | `-`/"" → Stdout | путь → File | 2 |
| `NewReportOutput` | json+Stdout | неизвестный формат → `ErrUnknownFormat` | 2 |

`Egress` (оркестратор-труба) и `ReportSink.Write` (I/O-труба) — **юнитами не
покрываются**. `buildErrorReport`/`exitCode` — уже покрыты.

Компонент (Gherkin): добавить **только режим отказа записи** — `--out <недоступный
путь>` → код 2 + `error.code=report_write_failed`. Матрица `stdout|файл × json|md`
остаётся юнитом и в `N = 1 + #extensions` не входит (граница со слоем юнитов).

## Решения по дизайну

- `WriteTo` удаляется целиком (не «оставить для тестов») — это и был боевой путь D1.
- Рендер вынесен из I/O-трубы в `internal/report` (чистая логика рядом с egress),
  `ReportSink` больше не знает форматов.
