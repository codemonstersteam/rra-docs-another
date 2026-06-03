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
| Реализация слайсов S1–S7 (E3–E9) | in progress (S1–S3, S5, S6 в main; S7 next) |

## Следующий шаг

**S4 `style` (L2) отложен в TBD** — внешние тулзы (Vale/markdownlint) не тянем,
состав L2 проектируем отдельно и научно (см. `backlog.md` → «Где мы сейчас»).

Влиты S1–S3 (L3, L1, L4), S5 `fitness` (L5, LLM), S6 `drift` (L6a,
детерминированный, `Judge`/`NoopJudge`, 24/24 компонентных зелёных). Конформанс с
ADR 0003 закрыт (E14, PR #9): словари L4, обязательные файлы L3 и манифесты L6
вынесены из хардкода в дефолтный конфиг.

Следующий — **E15** (рефактор: экспортный `Evaluate` на каждый слайс S1–S6),
**затем S7 `assess`**. Дизайн S7 утверждён (`docs/design/assess/slices/07-assess.md`):
**Option A** — `assess` добывает вход **1×** (`NewAuditTarget`+`NewConfig`+
`ReadStructure`, чьи `.Docs` кормят L1/L4/L5) и зовёт пять чистых листьев
`Evaluate`, а не пять голов (отклонённый Option B давал 5× валидацию/5× чтение).
Голова `ProcessAssess`: acquire → `layersUpTo` (`--up-to`) → `Evaluate` по плану →
`shortCircuit` («L4 FAIL → не звать L5») → условный L5 (LLM резолвится в голове;
нет ключа → `llm_unavailable`, код 2) → `mergeOutcomes`. Состав v1 = L1/L3/L4/L5/L6a
(**L2 нет** — S4 в TBD). Новых I/O нет, чеклист `http-io` не нужен. E15 — отдельным
PR до S7; реализация обоих — Sonnet.

Follow-up за S7 — **S8 `drift --semantic`** (тир L6c за флагом): `--semantic`
решается на краю (роутер выбирает реализацию `Judge`: реальный `LLMClient` /
`NoopJudge`), не ветвит голову `ProcessDrift`; перед S8 chore-PR — промоут
`LLMClient` `fitness/io.go` → `internal/io` (+ `Judge`); затем L6c по
`08-drift-semantic.md` и skill `http-io`.

Конвенция слайса (как в `ubik/passkey-demo-api`, см. `infrastructure.md`):
самодостаточный пакет `internal/slice/<name>/` — `head.go` (`Process<Slice>` —
голова), `adapter.go` (парсинг), `logic.go`, `register.go` (`Deps`+`NewDeps`).
Общие `internal/{domain,io,cli}` (egress в `cli`). Образцы — S1–S3, S5, S6 в main.

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
  эндпоинт через `--llm-base-url`). См. `rationaldev` ADR 0001. Резолвинг
  `baseURL`/`model`/`provider` — только в `domain.NewLLMConfig` (флаг > YAML > вшитый
  дефолт, anthropic → `/v1`); I/O-клиент берёт готовые значения, ничего не хардкодит.
- **Spec-first egress.** Любой исходящий HTTP к дозируемому сервису (LLM, будущие
  сервисы) проектируется по skill `http-io`: два бюджета (нагрузки/payload)
  считаются ДО кода; контракт проверяется curl-ом и замораживается машинной спекой
  в `api-specification/providers/<name>.openapi.yaml` (или AsyncAPI для стрима);
  из спеки выводятся клиент, стаб и фикстуры; формулы бюджетов юнитятся, контрактные
  ветки отказа — компонентом. LLM-специфика — skill `llm-client`. Дизайн-карта
  слайса с HTTP-вызовом проходит чеклист дизайна `http-io` до кода.
- CLI — stdlib (subcommand-switch в `internal/cli`), без cobra.
- Компонентные тесты — всегда в Docker Compose (объект = спецификация программы:
  сервис или CLI-тул). Для тула: бинарь = сервис compose против фикстур, внешний
  API (LLM) = заглушка-сервис в том же compose. httptest/in-process и «бинарь на
  хосте» не используем; развилку не пересматриваем. См. `skills/component-tests`.

## Открытые вопросы

- Имя бинаря (`rra-docs-another` длинновато — возможно укоротить).

## Фрейм работы с агентом

> Агент: прочитай `AGENTS.md` и `skills/` перед тем как отвечать.
