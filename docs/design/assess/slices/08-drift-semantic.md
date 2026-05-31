# S8 — drift --semantic (L6c семантический тир, LLM)

Вход: `CLI rra-docs-another drift <path> --semantic`. Не новая подкоманда —
расширение S6 за флагом. Новая интеграция: `LLMClient.Judge`. Тир **опциональный**
(follow-up после L6a). Спроектирован по skill `http-io` (исходящий HTTP к
дозируемому сервису) + `llm-client` (LLM-специфика).

## Идея

L6a (S6) берёт только механически проверяемое. L6c подключает утверждения, что
grep не берёт («README обещает ретрай на 503 с backoff»), **строго ограниченно**:
LLM судит **конкретную предъявленную пару** (сниппет доки + кусок кода) → OK/нет +
цитата. Запрещено «прочитай весь репозиторий» — это и есть граница payload-бюджета.

## Интеграция в голову — без `if` (см. `06-drift.md`)

L6c не добавляет ветку в `ProcessDrift`. Голова безусловна; `--semantic` выбирает
**реализацию** зависимости `Judge` в роутере:

- `NoopJudge{}` (null-object) — без флага: `Judge(set) -> ([], nil)`, ни ключа, ни сети;
- `LLMClient` — под флагом: `NewLLMConfig` (fail-fast по ключу) зовётся только здесь.

Тип зависимости — интерфейс `Judge`; обе реализации в `internal/io`.

## Контракт `LLMClient.Judge` по чеклисту `http-io`

Уточнение прежнего эскиза: пейсинг — ответственность I/O-объекта, поэтому **набор**,
а не одна пара:

```
Judge(ClaimPromptSet) -> Result<[]Verdict, Error>     # фан-аут пар внутри объекта
```

| Пункт чеклиста http-io | Решение |
|---|---|
| **Бюджет payload** | одна **пара** на вызов (сниппет доки `file:line ± окно` + кусок кода). Никогда не весь репозиторий. Пре-флайт `estimateTokens` на пару. |
| **Бюджет нагрузки** | N = число eligible-claims → **cap** `llm.max_judge_calls` (дефолт 20) в `buildClaimPromptSet` + пейсинг `call_delay_ms` + бэкофф по `Retry-After` (переиспользуем `retryWait`). `N × tokens/пара ≤ TPM`. Обрезка логируется. |
| **Протокол** | тот же OpenAI-совместимый `/chat/completions`; подключение — общий `LLMConfig`/`NewLLMConfig` (флаг > YAML > дефолт, `/v1`). |
| **response_format** | json_schema strict для `Verdict{ok bool, quote string}` (+ `additionalProperties:false`). |
| **Классы отказа** | те же sentinel: `ErrLLMRateLimited` / `ErrLLMUnavailable` / `ErrLLMBudgetExceeded` → `llm_*`. |
| **Спека провайдера** | расширить `api-specification/providers/anthropic-openai-compat.openapi.yaml` схемой `Verdict` (spec-first: клиент/стаб/фикстуры из неё). |
| **Стаб** | новый режим `judge` в `component-tests/llm-stub` (отдаёт `{ok,quote}`); сценарий `drift --semantic` снимается с `@wip`. |

## Промоут общего LLM-I/O (предусловие S8)

`LLMClient` сейчас слайс-локален (`internal/slice/fitness/io.go`). По
`infrastructure.md` он общий (S5/S7/S8) → **поднять в `internal/io/llmclient.go`** и
добавить `Judge` рядом с `Ask`. Чистые формулы-бюджеты (`estimateTokens`,
`overTokenBudget`, `retryWait`) — туда же, как переиспользуемые. Делается
отдельным chore-PR ДО реализации L6c (не смешивать с L6a S6).

## Контракты (чистая логика)

- `buildClaimPromptSet(check) [dep: Config] -> ClaimPromptSet` — пары по `Kind`, cap.
- `LLMClient.Judge(ClaimPromptSet) -> Result<[]Verdict, Error>` — I/O; ошибки → `llm_*`.
- `mergeSemanticFindings([]Verdict) -> []DriftFinding` — вердикт `OK=false` → finding+цитата.
- `NoopJudge.Judge(ClaimPromptSet) -> ([], nil)` — null-object.

## Сообщения

- `ClaimPrompt` `{claim Claim, docSnippet string, codeChunk string}` — пара на суд.
- `ClaimPromptSet` `{prompts []ClaimPrompt}` — вход `Judge`; конструктор
  `buildClaimPromptSet` (cap внутри).
- `Verdict` `{OK bool, Quote string}` — выход `Judge` на одну пару.

## Юнит-тесты

| Модуль | Happy | Ветки | Итого |
|---|---|---|---|
| buildClaimPromptSet | 1 | пусто (нет eligible); обрезка по cap | 3 |
| mergeSemanticFindings | 1 | вердикт OK=false → drift | 2 |

`Judge`/`NoopJudge` (I/O) — труба; happy и `llm_*`-отказы — компонентом
(`drift --semantic` через стаб-режим `judge`).

## Статус

`todo (follow-up)`. Реализуется после L6a (S6) и промоута общего LLM-I/O.
Дизайн зафиксирован; в реализацию L6a S6 не входит.
