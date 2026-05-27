# S7 — assess (весь пайплайн L1–L6)

Вход: `CLI rra-docs-another assess <path>`. Новых интеграций нет — собирает
пайплайн из **уже протестированных листьев** S1–S6 (не вызывает их головы).
Реализует short-circuit и правило «четыре независимых score, не усредняем».

## Дерево модулей

```
parseAssessArgs(args)                      -> Request
runAssess(req) [Deps: RepoStore, LinterRunner, LLMClient] -> (Report, error)
   | NewAuditTarget(req)                   -> AuditTarget
   | NewConfig(req)                        -> Config
   | layersUpTo(req)                       -> LayerPlan         # чистая логика (какие слои)
   | store.ReadStructure(target)           -> RepoStructure     # I/O
   | store.ReadMarkdownDocs(target)        -> []MarkdownDoc     # I/O
   | checkStructure(structure)             -> LayerOutcome (L3) # лист S1  [dep: Config]
   | scoreReadability(docs)                -> LayerOutcome (L1) # лист S2  [dep: Config]
   | buildJTBDCards(docs)                  -> []JTBDResult (L4) # листья S3 [dep: Config]
   | linter.Run(target) ⨾ aggregateFindings-> LayerOutcome (L2) # I/O S4 + лист S4
   | extractClaims ⨾ verifyClaims ⨾ buildDriftOutcome -> LayerOutcome (L6a) # листья S6
   | shortCircuit(jtbdL4)                  -> bool              # чистая логика
   | [если не short-circuit и план≥L5] L5: NewLLMConfig ⨾ buildJTBDPromptSet ⨾ llm.Simulate ⨾ scoreFitness -> []JTBDResult
   | buildReport(allParts)                 -> Report            # общий
```

## Псевдокод пайпа

```
runAssess(req) -> Result<Report, Error>:
    | NewAuditTarget(req)            -> AuditTarget
    | NewConfig(req)                 -> Config
    | layersUpTo(req)                -> LayerPlan
    | store.ReadStructure(target)    -> RepoStructure
    | store.ReadMarkdownDocs(target) -> []MarkdownDoc
    | дешёвые слои (по плану): L3 checkStructure, L1 scoreReadability,
    |   L4 buildJTBDCards, L2 (linter.Run+aggregateFindings), L6a (extract+verify+outcome)
    | shortCircuit(L4-результаты)    -> skipLLM: bool
    | если план≥L5 И не skipLLM:     L5 (NewLLMConfig+promptSet+Simulate+scoreFitness)
    | buildReport({Layers:[…], JTBD: L5 ?? L4}) -> Report
```

**Решение по дизайну.** Голова `assess` — единственная с управляющим потоком
(гейт `--up-to` и short-circuit «не звать LLM, если L4 упал»). Это не нарушение
«труба без ветвлений» из эскиза — это и есть логика интегратора. Решения вынесены
в чистые листья (`layersUpTo`, `shortCircuit`), сам пайп их применяет. Корректность
ветвлений доказывают компонентные сценарии (happy / bad-repo / отказы), не юниты
головы. Листья слоёв переиспользуются из S1–S6 (вызов листьев, не голов).

## Контракты модулей (новые для S7)

### layersUpTo
- **Сигнатура:** `layersUpTo(req Request) -> LayerPlan`
- **Input (data):** Request. **Dependencies:** —
- **Антецедент:** `req.UpTo ∈ {"", L1..L6}`. **Консеквент:** множество слоёв к
  исполнению (по умолчанию L1–L6; `--up-to L4` отсекает L5).

### shortCircuit
- **Сигнатура:** `shortCircuit(jtbdL4 []JTBDResult) -> bool`
- **Input (data):** []JTBDResult. **Dependencies:** —
- **Что делает:** возвращает `true`, если хотя бы один L4-результат `FAIL`
  (тогда L5/LLM не запускается — деньги не тратятся).
- **Консеквент:** `true` при наличии FAIL, иначе `false`.

(`buildReport`, листья S1–S6, I/O-объекты — описаны в своих карточках и
`infrastructure.md`. S7 их **не** дублирует и **не** перетестирует.)

## Юнит-тесты (только новые листья S7)

| Модуль | Happy | Ветки | Итого |
|---|---|---|---|
| layersUpTo | 1 | `--up-to L4` (отсечь L5) | 2 |
| shortCircuit | 1 | есть FAIL → true | 2 |

Голова `runAssess` — труба-интегратор, не юнитится. Листья L1–L6 уже покрыты в
S1–S6. `buildReport` покрыт в S1.

## Gherkin-mapping (`features/assess.feature`)

| Сценарий | Then-шаг | Кто обеспечивает |
|---|---|---|
| опрятный — четыре PASS | код возврата 0 | egress `exitCode` |
| опрятный — четыре PASS | `jtbd.{maintainer,consumer,manager,agent}.status` = "PASS" | L5 `scoreFitness` (или L4, если `--up-to L4`) → `buildReport` |
| битый — есть проваленный JTBD | код возврата 1 | egress `exitCode` (FAIL/blocker) |
| битый — есть проваленный JTBD | (short-circuit) | `shortCircuit` → L5 пропущен |
| путь не существует | код возврата 2 | egress ← `NewAuditTarget` |
| путь не существует | `errors[]` `path_not_found` | `buildErrorReport` |

[x] Gherkin-mapping сверен
