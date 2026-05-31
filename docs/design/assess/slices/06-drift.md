# S6 — drift (L6a дрейф документации, без ИИ)

Вход: `CLI rra-docs-another drift <path>`. Новых интеграций для L6a нет (`RepoStore`).
Детерминированно: извлечь механически проверяемые утверждения и сверить с ФС/кодом.
Семантический тир L6c — за флагом `--semantic` (S8); его LLM-I/O спроектирован в
`08-drift-semantic.md` по skill `http-io`. **Слайс — одна подкоманда с двумя тирами;
L6a реализуется первым, L6c — follow-up за флагом.**

## Принцип: голова без ветвления (`--semantic` решается на краю)

Skill `program-design`: голова — линейная труба, «никаких вложенных условий и
циклов в самом пайпе»; решения — на краю (адаптер/композиция). Поэтому
`--semantic` — это **выбор реализации I/O-зависимости `Judge`**, а не ветка в
голове:

- решение принимается **один раз в роутере** (`internal/cli`): без флага инжектится
  `NoopJudge` (null-object → `([], nil)`, **без ключа и сети**); с флагом — реальный
  `LLMClient` (его `NewLLMConfig` зовётся только под флагом → `drift` без `--semantic`
  ключа не требует);
- голова безусловна: `buildClaimPromptSet → Judge → mergeSemanticFindings`
  исполняются всегда; при null-object судья возвращает пусто → семантических находок нет;
- слияние L6a- и L6c-находок — узел-конструктор `NewDriftReport` (санкционированное
  место склейки двух data-аргументов), не `if`.

## Дерево модулей

```
internal/io/                         # I/O общий (промоут LLMClient из fitness)
  llmclient.go   LLMClient: Ask(JTBDPromptSet) + Judge(ClaimPromptSet)
  judge.go       type Judge interface { Judge(ClaimPromptSet) -> ([]Verdict, error) }
                 NoopJudge{}          # null-object: ([], nil) — тир L6c выключен

internal/slice/drift/
  adapter.go     parseDriftArgs(args) -> Request     (+ --semantic → Request.Semantic)
  head.go        ProcessDrift(req, Deps) -> (Report, error)   # линейная труба, без if
  logic.go       extractClaims · verifyClaims · buildClaimPromptSet ·
                 mergeSemanticFindings · buildDriftOutcome
  domain.go      Claim · DriftFinding · DriftCheck · DriftReport
                 (+ конструкторы NewDriftCheck, NewDriftReport)
  register.go    Deps{Store, Judge} · NewDeps(cfg, judge)     # judge инжектится извне
```

## Псевдокод пайпа (голова — без ветвления)

```
ProcessDrift(req, deps) -> Result<Report, Error>:
    | NewAuditTarget(req)                         -> AuditTarget        # ErrPathNotFound/ErrReadError
    | deps.Store.ReadStructure(target)            -> RepoStructure      [I/O]
    | extractClaims(structure)                    -> []Claim
    | NewDriftCheck(structure, claims)            -> DriftCheck         # бандл (один data-аргумент дальше)
    | verifyClaims(check)                         -> []DriftFinding     # L6a, механика
    | buildClaimPromptSet(check)                  -> ClaimPromptSet     # L6c-пары по Kind, capped; не по флагу
    | deps.Judge.Judge(promptSet)                 -> []Verdict          [I/O]  # null-object → []
    | mergeSemanticFindings(verdicts)             -> []DriftFinding     # L6c
    | NewDriftReport(l6aFindings, semFindings)    -> DriftReport        # узел-склейка (2 data-арг → один)
    | buildDriftOutcome(report)                   -> LayerOutcome
    | buildReport({Layers:[outcome]}, target, "drift") -> Report
```

Решение `--semantic` — в роутере, не в голове:

```
runDriftCmd(args, stdout, stderr):
    | req := drift.ParseArgs(args)
    | cfg := NewConfig(req)                         (err → egress config_invalid)
    | judge := io.NoopJudge{}                       # дефолт: тир выключен, ключ не нужен
    | если req.Semantic:                            # ← единственный if — на КРАЮ (выбор реализации)
    |     llmCfg := NewLLMConfig(req, cfg)          (err → egress llm_unavailable)
    |     judge = io.NewLLMClient(llmCfg, cfg.LLMCallDelayMs(), cfg.LLMTokenBudget(), cfg.LLMMaxRetries())
    | deps := drift.NewDeps(cfg, judge)
    | egress(ProcessDrift(req, deps), …)
```

## Объём claim-kinds (S6 v1)

`ReadStructure` даёт `Files`, `Docs.Lines`, `Manifests`. Дёшево и без новых I/O
проверяемы:

- **`link`/path** — относительные пути в backticks и fenced-блоках → резолв против
  `Files` (отлично от L3 `checkLinksResolve`, который берёт markdown-синтаксис
  `[text](url)`; здесь — утверждения в backticks/fenced/прозе, не дубль);
- **`dependency`** — заявленный стек/пакет → сверка против `Manifests` (go.mod и т.п.).

Откладываются (требуют чтения контента не-md файлов = новый I/O, ломает «ноль
интеграций» слайса): **`env`** (grep по коду), **`subcommand`**.

## Контракты модулей

### extractClaims
- **Сигнатура:** `extractClaims(structure RepoStructure) -> []Claim`
- **Input (data):** RepoStructure. **Dependencies:** —
- **Что делает:** вытаскивает пути в backticks и fenced-блоках (`link`) и заявленный
  стек (`dependency`); каждый `Claim` несёт `Kind` и `file:line`.
- **Консеквент:** `[]Claim` (kinds `link`|`dependency`).

### NewDriftCheck
- **Сигнатура:** `NewDriftCheck(structure RepoStructure, claims []Claim) -> DriftCheck`
- **Input:** конструктор-бандл. **Dependencies:** —
- **Консеквент:** `DriftCheck{structure, claims}` — один data-вход для verifyClaims/buildClaimPromptSet.

### verifyClaims
- **Сигнатура:** `verifyClaims(check DriftCheck) -> []DriftFinding`
- **Что делает:** для `link`-claim резолвит путь против `Files`; для `dependency`-claim
  ищет в `Manifests`. Нарушение — только при механическом подтверждении.
- **Консеквент:** `[]DriftFinding` L6a (битый путь, зависимость не в манифесте).

### buildClaimPromptSet
- **Сигнатура:** `buildClaimPromptSet(check DriftCheck) -> ClaimPromptSet` [dep: Config]
- **Что делает:** отбирает semantic-eligible claims (по `Kind`, не по флагу), для
  каждого собирает **пару** (сниппет доки `file:line ± окно` + кусок кода), обрезает
  до `cfg.MaxJudgeCalls` (бюджет нагрузки L6c). Лог при обрезке (no silent caps).
- **Консеквент:** `ClaimPromptSet` (вход `Judge`); пуст, если eligible-claims нет.

### mergeSemanticFindings
- **Сигнатура:** `mergeSemanticFindings(verdicts []Verdict) -> []DriftFinding`
- **Что делает:** вердикт `OK=false` → `DriftFinding` (с цитатой). Пустой вход → пусто.

### NewDriftReport
- **Сигнатура:** `NewDriftReport(l6a []DriftFinding, semantic []DriftFinding) -> DriftReport`
- **Что делает:** склейка двух источников находок в один data-объект (узел-конструктор).

### buildDriftOutcome
- **Сигнатура:** `buildDriftOutcome(report DriftReport) -> LayerOutcome`
- **Консеквент:** маппит находки в `Violation{layer:"L6"}`; `Status=fail` при
  blocker-дрейфе (битый путь/несуществующий target), иначе `pass`/`warn`.

(`NewAuditTarget`, `buildReport`, `store.ReadStructure` — см. S1. `Judge`/`NoopJudge`,
`ClaimPromptSet`, `Verdict`, `LLMClient.Judge` — см. `08-drift-semantic.md`.)

## Юнит-тесты

| Модуль | Happy | Ветки | Итого |
|---|---|---|---|
| extractClaims | 1 | нет утверждений | 2 |
| NewDriftCheck | 1 | — | 1 |
| verifyClaims | 1 | битый path; dependency не в манифесте | 3 |
| buildClaimPromptSet | 1 | пусто (нет eligible); обрезка по cap | 3 |
| mergeSemanticFindings | 1 | вердикт OK=false → finding; пусто | 3 |
| NewDriftReport | 1 | — | 1 |
| buildDriftOutcome | 1 | есть blocker → fail | 2 |

`Judge` (I/O) и `NoopJudge` юнитами не покрываются: happy/отказ — компонентом.

## Gherkin-mapping (`features/drift.feature`)

| Сценарий | Then-шаг | Кто обеспечивает |
|---|---|---|
| опрятный — дока согласована | код возврата 0 | egress `exitCode` |
| опрятный — дока согласована | `layers.L6.status` = "pass" | `buildDriftOutcome` |
| битый path — блокирующий дрейф | код возврата 1 | egress `exitCode` (blocker) |
| битый path — блокирующий дрейф | `layers.L6.status` = "fail" | `verifyClaims`→`buildDriftOutcome` |
| путь не существует | код 2 + `errors[]` `path_not_found` | egress ← `NewAuditTarget` |
| (S8) `drift --semantic` | happy + `llm_*`-отказы | `Judge` через стаб-режим `judge` |

L6a-сценарии (pass/fail/path_not_found) — снимаются с `@wip` при реализации S6.
Сценарий `--semantic` — добавляется при реализации L6c (см. `08-drift-semantic.md`).

[x] Gherkin-mapping сверен
