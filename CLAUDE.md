# CLAUDE.md — Контекст проекта для агента

## Проект

**rra-docs-another** — универсальный аудитор качества документации произвольного
git-репозитория для четырёх JTBD-потребителей; четыре независимых score +
пробелы. Шестислойная оценка L1–L6, ИИ только на L5 и опциональном L6c. **Не**
предполагает рациональную дисциплину (этим отличается от гейта `rra-docs`). Стек:
Go (CLI), Vale/markdownlint (L2), Anthropic API (L5/L6c). Концепция — `CONCEPT.md`,
план — `PLAN.md`.

## Статус модулей

| Модуль | Статус |
|--------|--------|
| Концепция (`CONCEPT.md`) | done |
| План (`PLAN.md`, `backlog.md`) | done |
| intent (`docs/intent.md`) | done |
| Каркас (E0) | done |
| Контракт + Gherkin (E1, гейт) | done (PR1 контракт + PR2 godog в main) |
| Проектный пакет (E2) | done (дизайн-PR влит = аппрув) |
| Реализация слайсов S1–S7 (E3–E9) | in progress (S1–S3 в main) |

## Следующий шаг

**S4 `style` (L2) отложен в TBD** — внешние тулзы (Vale/markdownlint) не тянем,
состав L2 проектируем отдельно и научно (см. `backlog.md` → «Где мы сейчас»).

Следующий — **S5 `fitness`** (L5, LLM) по `docs/design/assess/slices/05-fitness.md`,
ветка `feat/slice-fitness`. Дизайн утверждён с оператором: новый I/O `LLMClient`
(`Simulate(JTBDPromptSet) -> []LLMVerdict`); **проектный конфиг во внешнем YAML**
(`--config`, дефолт `go:embed`) — выносим `llm`-подключение и `prompts` ролей,
первая Go-зависимость `gopkg.in/yaml.v3` (ADR 0003); секретов в YAML нет
(`llm.api_key_env` = имя env-переменной). Загрузчик конфига — общая инфраструктура
в `internal/cli`. Компонент-харнесс готов — снять `@wip` с `fitness.feature`.
Реализация Sonnet'ом, голову `runFitness` уже сверили. Затем S7 `assess`, S6 `drift`.

Конвенция слайса (как в `ubik/passkey-demo-api`, см. `infrastructure.md`):
самодостаточный пакет `internal/slice/<name>/` — `head.go` (`Process<Slice>` —
голова), `adapter.go` (парсинг), `logic.go`, `register.go` (`Deps`+`NewDeps`).
Общие `internal/{domain,io,cli}` (egress в `cli`). Образцы — S1–S3 в main.
Дальше S5 `fitness` (LLM) → S6 `drift` → S7 `assess`. S8 (`drift --semantic`) — поздний.

## Принятые решения

- Один внешний вход = одна CLI-подкоманда = один slice (7 слайсов + поздний S8).
- Роль контракта для CLI: `api-specification/cli.md` + `report.schema.json`.
- Язык — Go (ради единого набора RRA), формулы L1 нативно; при недостаточной
  точности L1 — Python-сайдкар `ReadabilityRunner`. См. `docs/adr/0001-go-vs-python.md`.
- L6 = универсальный дрейф (L6a, без ИИ) + опциональный семантический тир L6c.
  Дисциплина-сверка (бывший L6b) сюда НЕ входит — она в гейте `rra-docs`.
- I/O изолирован в `RepoStore` / `LinterRunner` / `LLMClient` / `ReportSink`.
- Проектный конфиг (`--config`) — внешний YAML (дефолт через `go:embed`): словари
  L4, профиль L2, пороги, `llm`-подключение и `prompts` ролей L5. Первая Go-зависимость
  `gopkg.in/yaml.v3`. Секретов нет — `llm.api_key_env` указывает имя env-переменной,
  ключ из env. Загрузчик — общая инфраструктура в `internal/cli`. См. ADR 0003.
- LLM провайдер-агностичен: `anthropic` (дефолт) / `openai` (любой OpenAI-совм.
  эндпоинт через `--llm-base-url`). См. `rationaldev` ADR 0001.
- CLI — stdlib (subcommand-switch в `internal/cli`), без cobra.
- Компонентные тесты — всегда в Docker Compose (объект = спецификация программы:
  сервис или CLI-тул). Для тула: бинарь = сервис compose против фикстур, внешний
  API (LLM) = заглушка-сервис в том же compose. httptest/in-process и «бинарь на
  хосте» не используем; развилку не пересматриваем. См. `skills/component-tests`.

## Открытые вопросы

- Имя бинаря (`rra-docs-another` длинновато — возможно укоротить).

## Фрейм работы с агентом

> Агент: прочитай `AGENTS.md` и `skills/` перед тем как отвечать.
