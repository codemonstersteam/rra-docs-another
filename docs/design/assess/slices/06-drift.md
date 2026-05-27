# S6 — drift (L6a дрейф документации, без ИИ)

Вход: `CLI rra-docs-another drift <path>`. Новых интеграций нет (`RepoStore`).
Детерминированно: извлечь механически проверяемые утверждения и сверить с ФС/кодом.
Семантический тир L6c — поздний S8 за флагом `--semantic`.

## Дерево модулей

```
parseDriftArgs(args)                       -> Request
runDrift(req) [Deps: RepoStore]            -> (Report, error)
   | NewAuditTarget(req)                   -> AuditTarget
   | store.ReadStructure(target)           -> RepoStructure    # I/O (доки + манифесты + файлы)
   | extractClaims(structure)              -> []Claim          # чистая логика
   | NewDriftCheck(structure, claims)      -> DriftCheck        # конструктор-бандл (Шаг 3)
   | verifyClaims(check)                    -> []DriftFinding    # чистая логика
   | buildDriftOutcome(findings)           -> LayerOutcome
   | buildReport({Layers:[outcome]})       -> Report
```

## Псевдокод пайпа

```
runDrift(req) -> Result<Report, Error>:
    | NewAuditTarget(req)          -> AuditTarget
    | store.ReadStructure(target)  -> RepoStructure
    | extractClaims(structure)     -> []Claim
    | NewDriftCheck(structure, claims) -> DriftCheck
    | verifyClaims(check)          -> []DriftFinding
    | buildDriftOutcome(findings)  -> LayerOutcome
    | buildReport({Layers:[outcome]}) -> Report
```

`NewDriftCheck` — узел-конструктор: объединяет `RepoStructure` и `[]Claim` в одну
структуру, чтобы `verifyClaims` принимал ровно один data-аргумент.

## Контракты модулей

### extractClaims
- **Сигнатура:** `extractClaims(structure RepoStructure) -> []Claim`
- **Input (data):** RepoStructure. **Dependencies:** —
- **Что делает:** вытаскивает относительные ссылки/пути в backticks, fenced-команды,
  заявленный стек, env-переменные, список подкоманд — всё механически проверяемое.
- **Консеквент:** `[]Claim` с `Kind` и `file:line`.

### NewDriftCheck
- **Сигнатура:** `NewDriftCheck(structure RepoStructure, claims []Claim) -> Result<DriftCheck, Error>`
- **Input (data):** конструктор-бандл. **Dependencies:** —
- **Антецедент:** —. **Консеквент:** `DriftCheck{structure, claims}`.

### verifyClaims
- **Сигнатура:** `verifyClaims(check DriftCheck) -> []DriftFinding`
- **Input (data):** DriftCheck. **Dependencies:** —
- **Что делает:** для каждого `Claim` проверяет по ФС/манифестам; нарушение
  заявляется только при механическом подтверждении.
- **Консеквент:** `[]DriftFinding` (битая ссылка, несуществующий target, зависимость
  не в манифесте, env не найден grep'ом).

### buildDriftOutcome
- **Сигнатура:** `buildDriftOutcome(findings []DriftFinding) -> LayerOutcome`
- **Input (data):** []DriftFinding. **Dependencies:** —
- **Консеквент:** маппит находки в `Violation{layer:"L6"}`; `Status=fail` при
  blocker-дрейфе (битая ссылка/несуществующий target), иначе `pass`/`warn`.

(`NewAuditTarget`, `buildReport`, `store.ReadStructure` — см. S1.)

## Юнит-тесты

| Модуль | Happy | Ветки | Итого |
|---|---|---|---|
| extractClaims | 1 | нет утверждений | 2 |
| NewDriftCheck | 1 | — | 1 |
| verifyClaims | 1 | битая ссылка; зависимость не в манифесте | 3 |
| buildDriftOutcome | 1 | есть blocker → fail | 2 |

## Gherkin-mapping (`features/drift.feature`)

| Сценарий | Then-шаг | Кто обеспечивает |
|---|---|---|
| опрятный — дока согласована | код возврата 0 | egress `exitCode` |
| опрятный — дока согласована | `layers.L6.status` = "pass" | `buildDriftOutcome` (Success) |
| битая ссылка — блокирующий дрейф | код возврата 1 | egress `exitCode` (blocker present) |
| битая ссылка — блокирующий дрейф | `layers.L6.status` = "fail" | `verifyClaims`→`buildDriftOutcome` (Failure) |
| путь не существует | код возврата 2 | egress ← `NewAuditTarget` |
| путь не существует | `errors[]` `path_not_found` | `buildErrorReport` |

[x] Gherkin-mapping сверен
