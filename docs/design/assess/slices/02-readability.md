# S2 — readability (L1 читаемость)

Вход: `CLI rra-docs-another readability <path>`. Новых интеграций нет (только
`RepoStore` из S1). Чистая логика над `[]MarkdownDoc`.

**Особое правило L1:** порог-warning, не блок (CONCEPT §L1). Нарушения L1 всегда
`severity=warning`; статус слоя ∈ `{pass, warn}`. Команда `readability`
**никогда** не даёт код возврата 1.

## Дерево модулей

```
parseReadabilityArgs(args)            -> Request
runReadability(req) [Deps: RepoStore] -> (Report, error)
   | NewAuditTarget(req)              -> AuditTarget
   | NewConfig(req)                   -> Config              # пороги читаемости
   | store.ReadMarkdownDocs(target)   -> []MarkdownDoc       # I/O
   | scoreReadability(docs)           -> LayerOutcome        # [dep: Config]
       | pickFormula(doc)             -> fleschKincaid | obornevaRus
   | buildReport({Layers:[outcome]})  -> Report
```

`scoreReadability` опирается на чистые формулы-листья: `fleschKincaid` (англ.),
`obornevaRus` (рус., `FRE = 206.836 − 1.52·ASL − 65.14·ASW`, слоги ≈ гласные).
`pickFormula` выбирает формулу по доле кириллицы в тексте дока.

## Принятые решения (S2)

- **`buildReport`/`layerKey` — копируются** в `readability/head.go` дословно из S1
  (слайс самодостаточен; консолидация generic-сборки отчёта откладывается на S7
  `assess`). Утверждено оператором, 2026-05-27.
- **`domain.Config` дополняется** порогом читаемости: `ReadabilityMin() int`,
  дефолт `50` (по шкале FRE) в `NewConfig`.
- **`pickFormula(doc)`** — эвристика по языку: доля кириллицы ≥ 30% → `obornevaRus`,
  иначе `fleschKincaid`.

## Псевдокод пайпа

```
runReadability(req) -> Result<Report, Error>:
    | NewAuditTarget(req)            -> AuditTarget
    | NewConfig(req)                 -> Config
    | store.ReadMarkdownDocs(target) -> []MarkdownDoc
    | scoreReadability(docs)         -> LayerOutcome      # [dep: Config]
    | buildReport({Layers:[outcome]}) -> Report
```

## Контракты модулей

### fleschKincaid / obornevaRus
- **Сигнатура:** `(doc MarkdownDoc) -> float64`
- **Input (data):** MarkdownDoc. **Dependencies:** —
- **Антецедент:** —. **Консеквент:** оценка читаемости; для пустого текста —
  нейтральное значение (без паники, без деления на ноль).

### pickFormula
- **Сигнатура:** `pickFormula(doc MarkdownDoc) -> func(MarkdownDoc) float64`
- **Input (data):** MarkdownDoc. **Dependencies:** —
- **Антецедент:** —. **Консеквент:** `obornevaRus` при доле кириллицы ≥ 30%,
  иначе `fleschKincaid`; для пустого текста — `fleschKincaid` (без паники).

### scoreReadability
- **Сигнатура:** `scoreReadability(docs []MarkdownDoc, cfg Config) -> LayerOutcome`
- **Input (data):** []MarkdownDoc. **Dependencies:** `Config` (порог `ReadabilityMin`).
- **Что делает:** усредняет по докам, сравнивает с порогом, формирует L1-исход.
- **Консеквент:** `Status ∈ {pass, warn}` (никогда fail); `Violations` —
  только `warning` с `file:line` на сложных абзацах; `Score` 0–100.

(`NewAuditTarget`, `NewConfig`, `buildReport`, `store.ReadMarkdownDocs` — см. S1.)

## Юнит-тесты

| Модуль | Happy | Ветки | Итого |
|---|---|---|---|
| fleschKincaid | 1 | пустой текст | 2 |
| obornevaRus | 1 | пустой текст | 2 |
| pickFormula | 1 | кириллица ≥30% → rus; пустой → en | 3 |
| scoreReadability | 1 | низкая читаемость → warn (не fail) | 2 |

`NewAuditTarget`/`NewConfig` юнит-тесты учтены в S1 (один пакет `internal/domain`).

## Gherkin-mapping (`features/readability.feature`)

| Сценарий | Then-шаг | Кто обеспечивает |
|---|---|---|
| опрятный репозиторий | код возврата 0 | egress `exitCode` |
| опрятный репозиторий | `layers.L1.status` присутствует | `scoreReadability` → `buildReport` |
| низкая читаемость не блокирует | код возврата 0 | `scoreReadability` (warn, не blocker) + `exitCode` (L1-исключение) |
| путь не существует | код возврата 2 | egress ← `NewAuditTarget` (`ErrPathNotFound`) |
| путь не существует | `errors[]` `path_not_found` | `buildErrorReport` |

[x] Gherkin-mapping сверен
