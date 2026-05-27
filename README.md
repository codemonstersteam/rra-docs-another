# rra-docs-another

> Универсальный аудитор качества документации **произвольного** git-репозитория:
> оценивает пригодность для работы четырёх JTBD-потребителей и выдаёт четыре
> независимых score плюс список пробелов.

Часть продукта **Rational Repository Auditor (RRA)**, но позиционируется как
самостоятельный инструмент оценки качества. В отличие от узкого гейта `rra-docs`
(соответствие дисциплине), `rra-docs-another` работает на любом репо с гитхаба и
**не** предполагает рациональную дисциплину.

---

## Что умеет / НЕ умеет

**Умеет:**
- Шестислойная оценка L1–L6 (читаемость, стиль, структура, JTBD-присутствие,
  JTBD-пригодность через LLM, дрейф документации).
- Четыре независимых score по потребителям + пробелы с `file:line`.
- Markdown для человека и JSON для CI.

**НЕ умеет:**
- Не проверяет соответствие дисциплине из скиллов (это гейт `rra-docs`).
- Не усредняет JTBD; не линтер орфографии; не аудит I/O-границы кода.

## Стек

| Компонент | Технология |
|-----------|-----------|
| CLI | Go (stdlib) |
| Читаемость L1 | нативные формулы (Flesch-Kincaid, Оборнева) |
| Стиль L2 | Vale, markdownlint-cli2 (subprocess) |
| JTBD-пригодность L5 / семантический L6c | Anthropic API |

## CLI

```
rra-docs-another structure   <path>   # L3
rra-docs-another readability <path>   # L1
rra-docs-another jtbd        <path>   # L4
rra-docs-another style       <path>   # L2
rra-docs-another fitness     <path>   # L5 (LLM)
rra-docs-another drift       <path>   # L6 (ядро без ИИ; --semantic — LLM)
rra-docs-another assess      <path>   # весь пайплайн, дешёвое-первым
```

(Имя бинаря рабочее, может быть укорочено.)

Полный контракт — флаги, коды возврата, формат отчёта:
[`api-specification/cli.md`](./api-specification/cli.md) +
[`api-specification/report.schema.json`](./api-specification/report.schema.json).

## Карта режимов отказа

Весь внешний мир изолирован в I/O-объектах (`RepoStore`, `LinterRunner`,
`LLMClient`, `ReportSink`). Отказ любого из них означает, что **оценка не
выполнена** → код возврата `2` и запись в `errors[]` отчёта. В отличие от HTTP,
у CLI нет статусов и заголовков: режимы различаются полем `error.code` и
действием пользователя (код возврата у всех отказов одинаковый — `2`).

| Интеграция | `error.code` | Код | Действие пользователя |
|---|---|---|---|
| `RepoStore` (ФС/git) | `path_not_found` | 2 | проверить путь к репозиторию |
| `RepoStore` (ФС/git) | `read_error` | 2 | проверить права доступа / целостность файлов |
| — (конфиг) | `config_invalid` | 2 | исправить `--config` (синтаксис/схема) |
| `LinterRunner` (L2) | `tool_missing` | 2 | установить Vale / markdownlint-cli2 в `PATH` |
| `LinterRunner` (L2) | `tool_failed` | 2 | проверить профиль линтера и его вывод в stderr |
| `LLMClient` (L5/L6c) | `llm_rate_limited` | 2 | повторить с экспоненциальным backoff |
| `LLMClient` (L5/L6c) | `llm_unavailable` | 2 | проверить ключ в env, `--llm-base-url`, сеть; повторить позже |
| `LLMClient` (L5/L6c) | `llm_budget_exceeded` | 2 | поднять бюджет или сузить вход (`--up-to`, меньший репо) |

Коды возврата `0` (чисто) и `1` (есть блокирующий провал) означают, что оценка
**выполнена**; их семантика по командам — в [`api-specification/cli.md`](./api-specification/cli.md#коды-возврата).

## Документация

```
README.md (что это, как запустить)
  → CONCEPT.md (зачем и как устроена оценка L1–L6)
    → PLAN.md (план разработки маленькими шагами)
      → docs/concept-docs-readability.md (научная база L1–L3)
        → skills/ (дисциплина проектирования и реализации)
```

## Статус

Каркас (E0) готов; идёт E1 — контракт CLI и компонентные тесты. См. `backlog.md`.
