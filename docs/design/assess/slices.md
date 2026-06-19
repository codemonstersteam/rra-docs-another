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
| S7 | CLI | `rra-docs-another assess <path>` | assess | L1–L6 | — | `hasDocs`, `mergeOutcomes` (+ кэп `capL5ByL4`) |
| S8 | CLI | `rra-docs-another drift <path> --semantic` | drift+ | L6c | LLMClient | `buildClaimPrompt`, `judgePair` |

## Зависимости между слайсами

- S1 первым: ставит `RepoStore` + `ReportSink` + egress (формат отчёта, коды
  возврата) на простейшем слое L3.
- S2, S3, S6 — чистая логика над `[]MarkdownDoc` от `RepoStore` (ноль новых
  интеграций), могут идти после S1 в любом порядке.
- S4 добавляет `LinterRunner`.
- S5 добавляет `LLMClient` (единственный базовый LLM-слайс).
- S7 собирает пайплайн из **уже протестированных листьев** S1–S6 (не вызывает
  головы других слайсов), реализует гейт L5 `hasDocs` (статика не блокирует ИИ, а
  кэпит итог через `capL5ByL4`) и правило «четыре score».
- S8 — поздний, расширяет S6 семантическим тиром за флагом.

## Общая форма slice'а (CLI)

Слайс — самодостаточный пакет `internal/slice/<name>/` со строгим набором файлов
(`head.go`/`adapter.go`/`logic.go`/`domain.go`/`errors.go`/`register.go` — см.
`infrastructure.md`, конвенция как в `ubik/passkey-demo-api`). Голова — в `head.go`:

```
ParseArgs(args)                -> Request           # adapter.go: только парсинг
Process<Slice>(req, Deps)      -> (Report, error)   # head.go: головной модуль (пайп)
   | NewAuditTarget(req)       -> AuditTarget        # audit: валидация пути
   | NewConfig(req)            -> Config            # audit: валидация --config
   | deps.Store.Read…(target)  -> данные             # io: RepoStore
   | <чистые логика-листья>    -> LayerOutcome / []JTBDResult
   | buildReport(…)            -> Report            # report
```

Запись отчёта и код возврата — **общий egress** в `internal/report` (`Egress`),
не в голове: единая точка «результат → формат ответа» и для успеха (0/1), и для
отказа (`errors[]` + 2). Это осознанное отличие от per-endpoint mapError
HTTP-эталона — у тула отчёт машинно-единый (одна схема, один набор `error.code`).
Слайс подключает свою подкоманду сам через `register.go`.
