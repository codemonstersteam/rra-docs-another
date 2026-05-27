# messages — каталог сообщений assess

Структуры, которыми обмениваются модули. `Result<T, Error>` в Go = пара
`(T, error)` (дженерик-тип не вводим). Валидируемые доменные структуры имеют
**неэкспортируемые поля** и создаются только конструктором `NewT(...) -> (T, error)`.

## Невалидированный вход

### Request

Плоский DTO из ингресс-адаптера. Поля публичные, без правил домена. Один на все
слайсы (аргументы CLI однородны); адаптер заполняет релевантные подкоманде поля.

| Поле | Тип | Источник |
|---|---|---|
| `Command` | string | имя подкоманды |
| `Path` | string | позиционный `[path]` (по умолчанию `.`) |
| `Format` | string | `--format` (`md`\|`json`) |
| `Out` | string | `--out` (`-`\|файл) |
| `ConfigPath` | string | `--config` (может быть пустым) |
| `LLMProvider` | string | `--llm-provider` |
| `LLMBaseURL` | string | `--llm-base-url` |
| `LLMModel` | string | `--llm-model` |
| `UpTo` | string | `--up-to` (`assess`) |
| `Semantic` | bool | `--semantic` (`drift`) |

## Валидируемые доменные структуры (конструкторы)

### AuditTarget

Валидированный корень репозитория. Неэкспортируемые поля `root`, `commit`.

- `NewAuditTarget(req Request) -> (AuditTarget, error)`
- Антецедент: `req.Path` резолвится в существующую директорию с правами на чтение.
- Failure: `ErrPathNotFound` (нет пути / не директория), `ErrReadError` (нет прав).
- Success: `root` — абсолютный путь; `commit` — HEAD, если это git-репо, иначе `""`.

### Config

Валидированный проектный конфиг (словари заголовков L4, профиль линтеров L2,
пороги L1). Неэкспортируемые поля.

- `NewConfig(req Request) -> (Config, error)`
- Антецедент: если `req.ConfigPath != ""` — файл существует и парсится по схеме
  конфига; иначе берутся встроенные дефолты.
- Failure: `ErrConfigInvalid` (не читается / не по схеме).

### LLMConfig

Валидированная конфигурация LLM (для S5/S7/S8). Неэкспортируемые поля.

- `NewLLMConfig(req Request) -> (LLMConfig, error)`
- Антецедент: `provider ∈ {anthropic, openai}`; для `openai` `base_url` непустой;
  ключ присутствует в env (`ANTHROPIC_API_KEY`\|`OPENAI_API_KEY`).
- Failure: `ErrLLMUnavailable` (нет ключа / пустой base-url для openai).

## Данные (I/O-выход и промежуточные)

### MarkdownDoc

Выход `RepoStore.ReadMarkdownDocs`. Плоская структура (не доменный конструктор —
это считанные данные).

| Поле | Тип | Смысл |
|---|---|---|
| `Path` | string | путь относительно корня |
| `Lines` | []string | строки файла (для `file:line`) |
| `Headings` | []Heading | H1–H6: `{Level int, Text string, Line int}` |

### RepoStructure

Выход `RepoStore.ReadStructure`. Сырые факты ФС для L3/L6a.

| Поле | Тип |
|---|---|
| `Files` | []string (все файлы репо, относительные пути) |
| `Docs` | []MarkdownDoc |
| `MTimes` | map[string]time.Time |
| `Manifests` | map[string]string (go.mod/package.json/… → содержимое) |

### LayerOutcome

Единый результат слоя без ИИ-интеграций. Один выход на слой.

| Поле | Тип |
|---|---|
| `Result` | LayerResult `{Name string, Status string, Score *int, Summary string}` |
| `Violations` | []Violation |

`Status ∈ {pass, fail, warn, skipped}`; `Score` — указатель (`nil` = слой не даёт числа).

### JTBDResult

Результат по потребителю (L4 присутствие и L5 пригодность). Маппится в `jtbd.*`.

| Поле | Тип |
|---|---|
| `Consumer` | string (`maintainer`\|`consumer`\|`manager`\|`agent`) |
| `Status` | string (`PASS`\|`FAIL`\|`PARTIAL`) |
| `Score` | int (0–100) |
| `Gaps` | []string |

### JTBDPrompt (S5/S8)

Вход для `LLMClient`. Неэкспортируемые поля, конструктор `buildJTBDPrompt`.

| Поле | Тип | Смысл |
|---|---|---|
| `consumer` | string | роль |
| `budget` | int | бюджет токенов (из CONCEPT §L5) |
| `docs` | []MarkdownDoc | срез доков под роль |
| `questions` | []string | контрольные вопросы |

### Claim / DriftFinding (S6/S8)

- `Claim` `{Kind string, Text string, File string, Line int}` — извлечённое
  проверяемое утверждение (`link`\|`command`\|`dependency`\|`env`\|`subcommand`).
- `DriftFinding` `{Claim Claim, Reason string}` — утверждение, не подтверждённое
  репозиторием. Конвертируется в `Violation{layer:"L6", severity, file, line}`.

### Сборочные и I/O-выходные типы

- `StyleFindings` — выход `LinterRunner.Run`: `[]Finding{File string, Line int,
  Rule string, Severity string, Message string}`.
- `JTBDPromptSet` `{prompts []JTBDPrompt}` — набор из 4 промптов; конструктор
  `buildJTBDPromptSet(docs) [dep: Config] -> JTBDPromptSet`. Один вход
  `LLMClient.Simulate`.
- `LLMVerdict` `{Consumer string, RawStatus string, RawScore int, RawGaps
  []string}` — сырой провайдер-агностичный вердикт от `LLMClient.Simulate`;
  нормализуется в `JTBDResult` чистой `scoreFitness`.
- `DriftCheck` — бандл для L6a; конструктор `NewDriftCheck(structure, claims) ->
  DriftCheck` (узел-сборка по Шагу 3, чтобы `verifyClaims` имел один data-аргумент).
- `ReportParts` `{Layers []LayerOutcome, JTBD []JTBDResult}` — сборочный DTO,
  один вход общего `buildReport(parts) [dep: target, command] -> Report`.
- `ClaimPrompt` / `Verdict` (S8) — вход/выход `LLMClient.Judge`: предъявленная
  пара (сниппет доки + кусок кода) → `Verdict{OK bool, Quote string}`.

## Отчёт и провалы

### Report

Агрегат, сериализуется по `report.schema.json`. Чистая структура.

| Поле | Тип |
|---|---|
| `SchemaVersion` | string (`"1.0"`) |
| `Tool` | string (`"rra-docs-another"`) |
| `Command` | string |
| `Target` | `{Path string, Commit *string}` |
| `Layers` | map[string]LayerResult (ключи `L1`..`L6`) |
| `JTBD` | map[string]JTBDResult (ключи 4 потребителя) |
| `Violations` | []Violation |
| `Errors` | []Error |

### Violation

`{Code string, Layer string, Severity string, File string, Line *int, Message string}`.
`Severity ∈ {blocker, warning}`.

### Error

`{Code string, Integration *string, Message string}`. `Code` — один из восьми
`error.code` контракта. Создаётся egress'ом из доменной ошибки (см. таблицу ниже).

## Доменные ошибки → error.code

Типизированные sentinel-ошибки. Egress маппит их в `Error` и код возврата 2.

| Ошибка | `error.code` | Integration |
|---|---|---|
| `ErrPathNotFound` | `path_not_found` | RepoStore |
| `ErrReadError` | `read_error` | RepoStore |
| `ErrConfigInvalid` | `config_invalid` | — |
| `ErrToolMissing` | `tool_missing` | LinterRunner |
| `ErrToolFailed` | `tool_failed` | LinterRunner |
| `ErrLLMRateLimited` | `llm_rate_limited` | LLMClient |
| `ErrLLMUnavailable` | `llm_unavailable` | LLMClient |
| `ErrLLMBudgetExceeded` | `llm_budget_exceeded` | LLMClient |

## Транзитивная замкнутость

Все поля имеют объявленный тип; вложенные структуры (`Heading`, `LayerResult`,
`Violation`, `Error`, `JTBDResult`) описаны здесь. Валидируемые типы
(`AuditTarget`, `Config`, `LLMConfig`, `JTBDPrompt`) имеют конструктор
`NewT/buildT`. `TODO: уточнить тип` в каталоге нет.
