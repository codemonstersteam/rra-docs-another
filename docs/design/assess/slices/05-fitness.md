# S5 — fitness (L5 JTBD-пригодность, LLM)

Вход: `CLI rra-docs-another fitness <path>`. Новая интеграция: `LLMClient`
(единственный базовый LLM-слайс). Симуляция четырёх ролей; фан-аут инкапсулирован
в I/O-объект, чтобы в голове не было цикла.

## Дерево модулей

```
parseFitnessArgs(args)                     -> Request
runFitness(req) [Deps: RepoStore, LLMClient] -> (Report, error)
   | NewAuditTarget(req)                   -> AuditTarget
   | NewLLMConfig(req)                     -> LLMConfig        # провайдер + ключ из env
   | store.ReadMarkdownDocs(target)        -> []MarkdownDoc
   | buildJTBDPromptSet(docs)              -> JTBDPromptSet    # [dep: Config бюджеты/вопросы]
   | llm.Simulate(promptSet)               -> []LLMVerdict     # I/O: LLMClient (4 прогона внутри)
   | scoreFitness(verdicts)                -> []JTBDResult     # чистая нормализация
   | buildReport({JTBD: results})          -> Report
```

## Псевдокод пайпа

```
runFitness(req) -> Result<Report, Error>:
    | NewAuditTarget(req)            -> AuditTarget
    | NewLLMConfig(req)              -> LLMConfig
    | store.ReadMarkdownDocs(target) -> []MarkdownDoc
    | buildJTBDPromptSet(docs)       -> JTBDPromptSet     # [dep: Config]
    | llm.Simulate(promptSet)        -> []LLMVerdict
    | scoreFitness(verdicts)         -> []JTBDResult
    | buildReport({JTBD: results})   -> Report
```

`LLMConfig` строится **до** дорогого I/O: нет ключа → `ErrLLMUnavailable` сразу,
LLM не вызывается.

## Контракты модулей

### NewLLMConfig
- **Сигнатура:** `NewLLMConfig(req Request) -> Result<LLMConfig, Error>`
- **Input (data):** Request. **Dependencies:** — (env читается внутри).
- **Антецедент:** провайдер валиден; для `openai` base-url непустой; ключ в env.
- **Консеквент:** Success — `LLMConfig`. Failure — `ErrLLMUnavailable`.

### buildJTBDPromptSet
- **Сигнатура:** `buildJTBDPromptSet(docs []MarkdownDoc) -> JTBDPromptSet`
- **Input (data):** []MarkdownDoc. **Dependencies:** `Config` (бюджеты, вопросы).
- **Консеквент:** четыре `JTBDPrompt` с бюджетами из CONCEPT §L5 и срезом доков под роль.

### llm.Simulate (I/O)
- **Сигнатура:** `Simulate(set JTBDPromptSet) -> Result<[]LLMVerdict, Error>`
- **Input (data):** JTBDPromptSet. **Dependencies:** — (провайдер/ключ инкапсулированы).
- **Что делает:** четыре прогона LLM (anthropic native / openai-совм.), один
  режим взаимодействия (chat). Маппит ошибки любого провайдера в `llm_*`.
- **Консеквент:** Success — 4 `LLMVerdict`. Failure — `ErrLLMRateLimited` (429),
  `ErrLLMUnavailable` (сеть/5xx/нет ключа), `ErrLLMBudgetExceeded` (учёт токенов).

### scoreFitness
- **Сигнатура:** `scoreFitness(verdicts []LLMVerdict) -> []JTBDResult`
- **Input (data):** []LLMVerdict. **Dependencies:** —
- **Что делает:** нормализует сырые вердикты в `JTBDResult` (валидирует статус/score).
- **Консеквент:** четыре `JTBDResult`; некорректный сырой статус → консервативный `PARTIAL`.

(`NewAuditTarget`, `buildReport`, `store.ReadMarkdownDocs` — см. S1.)

## Юнит-тесты

| Модуль | Happy | Ветки | Итого |
|---|---|---|---|
| NewLLMConfig | 1 | нет ключа; openai без base-url | 3 |
| buildJTBDPromptSet | 1 | пустые доки | 2 |
| scoreFitness | 1 | некорректный сырой статус → PARTIAL | 2 |

`llm.Simulate` — труба, не юнитится: success зеленит happy-сценарий (стаб
`healthy`), failure-ветки — сценарии отказа (стаб `rate_limited`/`unavailable`/
`budget_exceeded`).

## Gherkin-mapping (`features/fitness.feature`)

| Сценарий | Then-шаг | Кто обеспечивает |
|---|---|---|
| стаб `healthy` — четыре PASS | код возврата 0 | egress `exitCode` |
| стаб `healthy` — четыре PASS | `command`="fitness" | `buildReport` |
| стаб `healthy` — четыре PASS | `jtbd.{maintainer,consumer,manager,agent}.status`="PASS" | `llm.Simulate`→`scoreFitness`→`buildReport` (фан-аут в 4 секции) |
| стаб `healthy` — четыре PASS | `jtbd.maintainer.score` непустой, `.gaps` присутствует | `scoreFitness` (поля схемы `jtbdResult`) |
| стаб `mixed` — score независимы | код возврата 1 | egress `exitCode` (JTBD `FAIL` → 1) |
| стаб `mixed` — score независимы | `jtbd.agent.status`="FAIL", `.gaps` непустой | `scoreFitness` (gaps протекают) |
| стаб `mixed` — score независимы | `jtbd.consumer.status`="PARTIAL", `maintainer`/`manager`="PASS" | не-усреднение: провал роли не тянет другие |
| LLM ограничивает частоту | код 2 + `errors[]` `llm_rate_limited` integration `LLMClient` | egress ← `llm.Simulate` (`ErrLLMRateLimited`) |
| LLM недоступен | код 2 + `errors[]` `llm_unavailable` integration `LLMClient` | `buildErrorReport` ← `ErrLLMUnavailable` |
| бюджет превышен | код 2 + `errors[]` `llm_budget_exceeded` integration `LLMClient` | `buildErrorReport` ← `ErrLLMBudgetExceeded` |
| ключ не задан в env — LLM не вызывается | код 2 + `errors[]` `llm_unavailable` integration `LLMClient` | `NewLLMConfig` (fail-fast до I/O) |
| битый `--config` | код 2 + `errors[]` `config_invalid` | загрузчик конфига (`internal/cli`) |

[x] Gherkin-mapping сверен

## Контракт стаб ↔ дефолтные промпты (для компонент-тестов)

`llm-stub` различает вердикт по роли, отыскивая в теле запроса маркер
**`role:<key>`** (`key ∈ maintainer|consumer|manager|agent`). Поэтому **каждый
дефолтный (вшитый `go:embed`) промпт обязан нести свой `role:<key>`** — иначе стаб
не различит роли и фан-аут в четыре секции не специфицируется. Реальный провайдер
маркер игнорирует; стаб реагирует на содержимое промпта детерминированно, как
реагировала бы модель. Режимы стаба: `healthy` (все PASS, разные score),
`mixed` (consumer PARTIAL, agent FAIL — независимость и не-усреднение),
`rate_limited`/`unavailable`/`budget_exceeded` (режимы отказа).

## Решения по дизайну (подключение и промпты — внешний YAML)

См. `docs/adr/0003-yaml-config.md`.

- **Подключение LLM и промпты четырёх ролей выносятся в проектный YAML** (`--config`,
  дефолт вшит через `go:embed`), чтобы промпты дорабатывались без пересборки.
  Зависимость `gopkg.in/yaml.v3` — первая в проекте, только на парсинг конфига.
- **Загрузчик конфига — общая инфраструктура**, не часть слайса: читается на
  бутстрапе в `internal/cli` (I/O на краю), парсится в value-конфиг и инжектится —
  `llm`-секция в `NewLLMClient(conn)`, `prompts`-секция в `Deps`/`Config` слайса
  (вход для `buildJTBDPromptSet`). Битый файл → `config_invalid`.
- **Секретов в YAML нет.** `llm.api_key_env` указывает лишь *имя* env-переменной с
  ключом (по умолчанию `ANTHROPIC_API_KEY`/`OPENAI_API_KEY`); сам ключ читается из
  env в `NewLLMConfig`. Нет ключа в указанной env → `ErrLLMUnavailable` до I/O.
- **Приоритет значений:** флаг `--llm-*` > `--config` > вшитый дефолт.
- `buildJTBDPromptSet(docs, cfg)` берёт промпты и бюджеты из загруженного конфига,
  а не из хардкода — иначе голова/листья не меняются.
