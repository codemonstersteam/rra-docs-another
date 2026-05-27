# slices — assess

Один внешний вход (CLI-подкоманда) = один slice. Тип входа у всех — `CLI`.
Порядок = дешёвое-и-ценное первым (см. `PLAN.md`).

| # | Тип | Идентификатор (подкоманда) | Slice | Слой | Новые интеграции | Логика-листья |
|---|-----|----------------------------|-------|------|------------------|----------------|
| S1 | CLI | `rra-docs-another structure <path>` | structure | L3 | RepoStore, ReportSink | `checkStructure` |
| S2 | CLI | `rra-docs-another readability <path>` | readability | L1 | — | `fleschKincaid`, `obornevaRus`, `scoreReadability` |
| S3 | CLI | `rra-docs-another jtbd <path>` | jtbd | L4 | — | `matchHeadings`, `buildJTBDCard` (×4) |
| S4 | CLI | `rra-docs-another style <path>` | style | L2 | LinterRunner | `aggregateFindings` |
| S5 | CLI | `rra-docs-another fitness <path>` | fitness | L5 | LLMClient | `buildJTBDPrompt`, `scoreFitness` |
| S6 | CLI | `rra-docs-another drift <path>` | drift | L6a | — | `extractClaims`, `verifyAgainstRepo` |
| S7 | CLI | `rra-docs-another assess <path>` | assess | L1–L6 | — | `shortCircuit`, `mergeReport` |
| S8 | CLI | `rra-docs-another drift <path> --semantic` | drift+ | L6c | LLMClient | `buildClaimPrompt`, `judgePair` |

## Зависимости между слайсами

- S1 первым: ставит `RepoStore` + `ReportSink` + egress (формат отчёта, коды
  возврата) на простейшем слое L3.
- S2, S3, S6 — чистая логика над `[]MarkdownDoc` от `RepoStore` (ноль новых
  интеграций), могут идти после S1 в любом порядке.
- S4 добавляет `LinterRunner`.
- S5 добавляет `LLMClient` (единственный базовый LLM-слайс).
- S7 собирает пайплайн из **уже протестированных листьев** S1–S6 (не вызывает
  головы других слайсов), реализует short-circuit и правило «четыре score».
- S8 — поздний, расширяет S6 семантическим тиром за флагом.

## Общая форма slice'а (CLI)

Все слайсы делят одну форму:

```
parse<Slice>Args(args)         -> Request          # ингресс-адаптер: только парсинг
run<Slice>(req) [deps]         -> (Report, error)  # головной модуль (пайп)
   | NewAuditTarget(req)       -> AuditTarget       # конструктор: валидация пути
   | NewConfig(req)            -> Config            # конструктор: валидация --config
   | store.Read…(target)       -> данные             # I/O: RepoStore
   | <чистые логика-листья>    -> LayerOutcome / []JTBDResult
   | buildReport(…)            -> Report            # чистая логика
```

Запись отчёта (`ReportSink.Write`) и вычисление кода возврата вынесены в **общий
egress** (`internal/cli`, см. `infrastructure.md`), а не в голову каждого слайса:
он одинаково форматирует и успех (код 0/1), и отказ (`errors[]` + код 2). Это
осознанное уточнение CLI-эгресса относительно эскиза S2 в скилле — единая точка
маппинга «результат → формат ответа».
