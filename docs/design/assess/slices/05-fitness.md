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
| стаб отвечает | код возврата 0 | egress `exitCode` |
| стаб отвечает | `jtbd.maintainer.status` = "PASS" | `llm.Simulate`→`scoreFitness`→`buildReport` |
| LLM ограничивает частоту | код возврата 2 | egress ← `llm.Simulate` (`ErrLLMRateLimited`) |
| LLM ограничивает частоту | `errors[]` `llm_rate_limited` | `buildErrorReport` |
| LLM недоступен | `errors[]` `llm_unavailable` | `buildErrorReport` ← `ErrLLMUnavailable` |
| бюджет превышен | `errors[]` `llm_budget_exceeded` | `buildErrorReport` ← `ErrLLMBudgetExceeded` |

[x] Gherkin-mapping сверен
