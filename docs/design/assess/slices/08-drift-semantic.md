# S8 — drift --semantic (L6c семантический тир, LLM) — ПОЗДНИЙ

Вход: `CLI rra-docs-another drift <path> --semantic`. Не новая подкоманда —
расширение S6 за флагом. Новая интеграция здесь: `LLMClient.Judge`. Тикет
**поздний и опциональный** (после S1–S7). Карточка — эскиз для будущей детализации.

## Идея

L6a (S6) берёт только механически проверяемое. L6c подключает утверждения, что
grep не берёт («README обещает ретрай на 503 с backoff»), строго ограниченно:
LLM **судит конкретную предъявленную пару** (сниппет доки + кусок кода) → Y/N +
цитата. Запрещено «прочитай весь репозиторий».

## Дерево модулей (эскиз)

```
runDrift(req) [+ если req.Semantic]:
   | … L6a как в S6 …
   | selectSemanticClaims(claims)     -> []Claim          # что не взял L6a
   | buildClaimPrompt(claim) ⨾ llm.Judge(prompt) -> Verdict  # I/O: LLMClient (по паре)
   | mergeSemanticFindings(verdicts)  -> []DriftFinding
```

## Контракты (эскиз)

- `buildClaimPrompt(claim Claim) -> ClaimPrompt` [dep: Config] — собирает пару.
- `llm.Judge(prompt ClaimPrompt) -> Result<Verdict, Error>` — I/O; ошибки → `llm_*`.
- `judgePair`/`mergeSemanticFindings([]Verdict) -> []DriftFinding` — чистая логика.

## Юнит-тесты (эскиз)

| Модуль | Happy | Ветки | Итого |
|---|---|---|---|
| buildClaimPrompt | 1 | — | 1 |
| mergeSemanticFindings | 1 | вердикт N → drift | 2 |

`llm.Judge` — труба; happy и `llm_*`-отказы покрываются `@wip`-сценарием
`drift --semantic` (добавляется при детализации S8).

## Статус

`todo (поздний)`. Детализируется отдельной итерацией program-design перед
реализацией; в основной хендофф S1–S7 не входит.
