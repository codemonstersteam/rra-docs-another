# S7 — assess (пайплайн L1, L3, L4, L5, L6a)

Вход: `CLI rra-docs-another assess <path>`. Новых интеграций нет. Собирает аудит за
**один проход**: добывает вход однажды и зовёт чистые **листья-оценки** слоёв
S1–S6 (`Evaluate`), сливая их в один `Report`. Реализует план `--up-to`,
гейт L5 «есть документация → зовём LLM» (`hasDocs`), ограничение итога статикой
(`capByL4`: FAIL на L4 кэпит L5 сверху до PARTIAL) и правило «четыре независимых
JTBD-score, не усредняем».

> **L2 (style) в пайплайне НЕТ.** S4 отложен в TBD (внешние тулзы не тянем,
> `LinterRunner` в коде отсутствует). Состав слоёв S7 v1 = **L1, L3, L4, L5, L6a**.
> Когда S4 появится, L2 добавляется в план и в merge без смены формы головы.

## Решение по reuse (Option A, утв. оператором)

S7 зовёт **листья поверх уже прочитанных данных**, не головы. Каждый слайс S1–S6
выставляет один экспортный вход `Evaluate(<данные>, cfg[, dep]) -> Outcome`; его
собственная голова делегирует туда же. `assess` добывает вход **1 раз** и зовёт
пять `Evaluate`. Предусловие — рефактор **E15** (отдельный PR до этой реализации).

**Почему не Option B (зовём пять голов, мержим `Report`'ы):** каждая голова
самодостаточна и внутри себя делает `NewAuditTarget` + `NewConfig` + чтение репы →
на полном прогоне 5× валидация и 5× чтение ФС *на каждом запуске аудита*. Option A
платит разовую цену рефактора S1–S6, но убирает дублирование под корень.

**Ключ к «1× чтение»:** `RepoStore.ReadStructure` уже возвращает
`RepoStructure{Files, Docs, Manifests, MTimes}`, где `Docs` = те же
`[]MarkdownDoc`, что нужны L1/L4/L5. Один `ReadStructure` кормит все пять слоёв.

## Решение по L5 без ключа (утв. оператором)

Дефолтный `assess` **требует** LLM на L5: нет ключа в env → `ErrLLMUnavailable` →
`error.code=llm_unavailable`, код 2. Прогон без ключа — только явным `--up-to L4`
(тогда L5 вне плана, ключ не нужен). Поведение совпадает со standalone `fitness`.

## Экспортные листья-оценки (из E15)

| Слой | Лист-оценка | Внутри | Dep |
|---|---|---|---|
| L1 | `readability.Evaluate(docs, cfg) LayerOutcome` | `scoreReadability` | — |
| L3 | `structure.Evaluate(s, cfg) LayerOutcome` | `checkStructure` | — |
| L4 | `jtbd.Evaluate(docs, cfg) map[string]JTBDResult` | `matchHeadings`+`buildJTBDCard`×N | — |
| L5 | `fitness.Evaluate(docs, cfg, llm) (map[string]JTBDResult, error)` | `buildJTBDPromptSet`→`Ask`→`scoreFitness` | `LLMClient` |
| L6a | `drift.Evaluate(s, cfg, judge) (LayerOutcome, error)` | `extractClaims`→`verifyClaims`→`buildClaimPromptSet`→`judge`→`buildDriftOutcome` | `Judge` (Noop) |

`docs` для всех слоёв = `s.Docs` из одного `ReadStructure`. Фильтр `cfg.Docs()` в
`fitness.Evaluate` — in-memory по `docs`.

## Дерево модулей

```
ParseArgs(args)                         -> Request        # adapter.go (+ --up-to)
ProcessAssess(req, deps) [Deps: Store, Judge] -> (Report, error)   # head.go — интегратор
   | NewAuditTarget(req)                -> AuditTarget      # 1× валидация пути
   | NewConfig(req)                     -> Config          # 1× валидация конфига
   | deps.Store.ReadStructure(t, cfg.Manifests()) -> RepoStructure  # 1× чтение ФС
   | layersUpTo(req.UpTo)               -> LayerPlan        # logic.go: какие слои
   | --- детерминированные слои по плану (дёшево-первым) ---
   | if plan[L1]: l1 := readability.Evaluate(s.Docs, cfg)
   | if plan[L3]: l3 := structure.Evaluate(s, cfg)
   | if plan[L4]: l4 := jtbd.Evaluate(s.Docs, cfg)
   | if plan[L6]: l6, err := drift.Evaluate(s, cfg, deps.Judge)   # Noop → err=nil
   | --- L5 условно: план≥L5 И есть документация ---
   | if plan[L5] && hasDocs(s.Docs):     # logic.go: len(docs)>0
   |     llmCfg, err := domain.NewLLMConfig(req, cfg)      # нет ключа → err → код 2
   |     l5, err := fitness.Evaluate(s.Docs, cfg, newLLM(cfg, llmCfg))  # llm_* → код 2
   | --- слияние (mergeOutcomes кэпит l5 по l4: FAIL L4 → PASS L5 → PARTIAL) ---
   | mergeOutcomes(out{l1,l3,l4,l5,l6}, plan, t) -> Report  # logic.go (семантика ниже)
```

## Псевдокод головы

```
ProcessAssess(req, deps) -> Result<Report, Error>:
    | t,   err := NewAuditTarget(req);              если err: return err   # код 2
    | cfg, err := NewConfig(req);                   если err: return err   # код 2
    | s,   err := deps.Store.ReadStructure(t, cfg.Manifests()); если err: return err
    | plan := layersUpTo(req.UpTo)
    | out  := {}                                    # LayerOutcome / JTBD по слоям
    | если L1 ∈ plan: out.L1 = readability.Evaluate(s.Docs, cfg)
    | если L3 ∈ plan: out.L3 = structure.Evaluate(s, cfg)
    | если L4 ∈ plan: out.L4 = jtbd.Evaluate(s.Docs, cfg)
    | если L6 ∈ plan: out.L6, err = drift.Evaluate(s, cfg, deps.Judge); если err: return err
    | если L5 ∈ plan И hasDocs(s.Docs):
    |     llmCfg, err := NewLLMConfig(req, cfg);    если err: return err   # llm_unavailable
    |     out.L5, err = fitness.Evaluate(s.Docs, cfg, newLLM(cfg, llmCfg)); если err: return err
    | return mergeOutcomes(out, plan, t)           # merge кэпит L5 по L4 (FAIL→PARTIAL)
```

**Решение по дизайну.** Голова `assess` — единственная с управляющим потоком (план
`--up-to`, гейт `hasDocs`, условный резолв LLM). Это логика интегратора, а не
нарушение «трубы»: решения вынесены в чистые листья (`layersUpTo`, `hasDocs`,
`capByL4`), пайп их применяет. Корректность ветвлений доказывают компонентные
сценарии (happy / bad-repo / отказ), не юниты головы.

**Деривация vs других слайсов:** `assess` — единственная команда, резолвящая
`NewLLMConfig` **внутри головы** (условно, после плана + гейта `hasDocs`): ключ нужен
лишь если реально доходим до L5. У `fitness`/`drift --semantic` резолв — в cli
(fail-fast). Поэтому `assess.Deps` несёт только `Store` и `Judge`(=Noop); `llmCfg`
и LLM-клиент строятся в голове по ветке L5.

## Семантика merge (`mergeOutcomes`)

Вход: `LayerPlan`, `AuditTarget`, исполненные оценки слоёв. Выход — единый
`Report{Command:"assess"}`:

- **`layers`** ← детерминированные слои: `L1` (readability), `L3` (structure),
  `L6` (drift). Слои, **отсечённые планом** (`--up-to`), пишутся маркером
  `layerResult{status:"skipped"}` (схема допускает `skipped`). L4/L5 в `layers`
  не выносим — их результат живёт в `jtbd` (контракт `cli.md`).
- **`jtbd`** (четыре потребителя) ← из L5 (`fitness`), если L5 исполнялся; **иначе**
  из L4 (`jtbd`-presence). Правило `L5 ?? L4`. НЕ усредняется. Если план < L4 —
  секции `jtbd` нет (допустимо схемой). **Кэп статикой:** когда берём L5, каждый
  статус ограничивается сверху результатом L4 той же роли — для роли с `L4=FAIL`
  итог `PASS` понижается до `PARTIAL` (приватный лист `capL5ByL4` внутри
  `mergeOutcomes`; `FAIL`/`PARTIAL` L5 и Score/Gaps не переписываются). Слияние
  двух карт идёт внутри агрегатора `layerOutcomes` — отдельного двухаргументного
  узла в графе нет (правило одного data-аргумента).
- **`violations`** ← конкатенация `Violations` из `LayerOutcome` исполненных слоёв
  (коды/слои уже проставлены листьями; `assess` ничего не переписывает).
- **`target`/`schema_version`/`tool`** — общие; `command="assess"`.

Код возврата — общий `exitCode` (egress): `Errors`→2; blocker-`violation`
(не L1) или JTBD `FAIL`→1; иначе 0. `assess` своей логики кода не имеет.

## Контракты новых модулей (только новое в S7)

### layersUpTo
- **Сигнатура:** `layersUpTo(upTo string) -> LayerPlan`
- **Антецедент:** `upTo ∈ {"", L1..L6}` (валидность гарантирует `ParseArgs`).
- **Консеквент:** множество слоёв из существующих {L1,L3,L4,L5,L6}, порядок
  L1<L2<L3<L4<L5<L6; `Lk` = все существующие слои ≤ k. `""`=L6 (все).
  Примеры: `""`/`L6`→{L1,L3,L4,L5,L6}; `L4`→{L1,L3,L4}; `L1`→{L1}.

### hasDocs
- **Сигнатура:** `hasDocs(docs []MarkdownDoc) -> bool`
- **Антецедент:** `docs` = `s.Docs` из одного `ReadStructure`.
- **Консеквент:** `true`, если документация присутствует (`len(docs) > 0`) — тогда
  L5/LLM запускается. Пустой набор → `false` (оценивать нечего, L5 пропускается).

### mergeOutcomes
- **Сигнатура:** `mergeOutcomes(out layerOutcomes, plan LayerPlan, target AuditTarget) -> Report`
- **Что делает:** слияние по семантике выше (layers L1/L3/L6 + skipped-маркеры,
  jtbd = L5 ?? L4 **с кэпом статикой**, violations∪). Один data-аргумент —
  агрегатор `layerOutcomes` (держит l1/l3/l4/l5/l6); `plan`/`target` — контекст.
  Чистая функция, без I/O.
- **Кэп (приватный лист `capL5ByL4`):** при формировании `jtbd` из L5 для каждой
  роли с `l4[role].Status == "FAIL"` статус `PASS` понижается до `PARTIAL`;
  `FAIL`/`PARTIAL` L5 сохраняются; роли без `FAIL` на L4 (или `l4 == nil`) — без
  изменений; Score/Gaps L5 не переписываются. Внутренний помощник `mergeOutcomes`,
  не узел графа и не шаг головы.

(`NewAuditTarget`/`NewConfig`/`NewLLMConfig`, листья `Evaluate` S1–S6, I/O-объекты,
egress — описаны в своих карточках, `infrastructure.md`, `messages.md` и в E15. S7
их **не** дублирует и **не** перетестирует.)

## Adapter (`ParseArgs`)

Общие флаги (`--format`/`--out`/`--config`/`--llm-*`) + специфичный `--up-to`
(`L1..L6`, дефолт пусто=L6). Невалидное `--up-to` → ошибка парсинга → код 2.
`--semantic` к `assess` не относится (флаг drift): L6a в `assess` всегда с `NoopJudge`.

## Юнит-тесты (только новые листья S7)

| Модуль | Happy | Ветки | Итого |
|---|---|---|---|
| layersUpTo | дефолт→{L1,L3,L4,L5,L6} | `L4`→отсечь L5,L6 | 2 |
| hasDocs | есть docs→true | пустой набор→false | 2 |
| capL5ByL4 (приватный лист merge) | FAIL L4 + PASS L5 → PARTIAL | PASS L4 → L5 без изменений; FAIL/PARTIAL L5 не трогаем; `l4==nil` → noop | 3 |
| mergeOutcomes | L5 исполнен → jtbd из L5 (с кэпом), layers L1/L3/L6 | L5 вне плана → jtbd из L4 + маркер skipped; violations∪ | 2–3 |

Голова `ProcessAssess` — труба-интегратор, не юнитится (корректность — компонентом).
Листья `Evaluate` L1/L3/L4/L5/L6a покрыты юнитами в своих слайсах (E15).

## Gherkin-mapping (`features/assess.feature`)

| Сценарий | Then-шаг | Кто обеспечивает |
|---|---|---|
| опрятный — четыре PASS | код 0 | egress `exitCode` |
| опрятный — четыре PASS | `jtbd.{maintainer,consumer,manager,agent}.status`="PASS" | `fitness.Evaluate` → `mergeOutcomes` (jtbd←L5) |
| docs есть, статика частично провалена, ИИ годен | код 0 | egress `exitCode` (нет FAIL: merge кэпит → PARTIAL) |
| docs есть, статика частично провалена, ИИ годен | `jtbd.agent.status`="PARTIAL" | `hasDocs`(L5 запущен) → `mergeOutcomes`/`capL5ByL4`(FAIL L4 кэпит PASS L5) |
| битый — ИИ подтверждает провал | код 1 | `hasDocs`(L5 запущен) → `fitness.Evaluate`(FAIL) → egress `exitCode` (JTBD FAIL) |
| путь не существует | код 2 | `NewAuditTarget` → `buildErrorReport` |
| путь не существует | `errors[]` `path_not_found` | `mapError` |

Дополнительно (E1.1, отдельный PR обвязки): `assess --up-to L4` →
`layers.L5.status` отсутствует (L5 в jtbd), `jtbd` из L4, ключ LLM не требуется.

[x] Gherkin-mapping сверен
[x] reuse-стратегия (Option A) и L5-без-ключа утверждены оператором
[x] предусловие E15 (экспортные `Evaluate`) заведено в backlog
