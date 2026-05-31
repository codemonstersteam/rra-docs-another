# contracts-graph — assess

Сверка согласованности контрактов (Шаг 9). Один граф на слайс + проверка
транзитивной замкнутости каталога сообщений. Зависимости (`*os`, `*exec`,
`*http`, env) на стрелках не показываются — они скрыты в I/O-объектах.

## 9.1. Каталог сообщений — транзитивная замкнутость

Пройден `messages.md`: у каждого поля объявлен тип; вложенные структуры
(`Heading`, `LayerResult`, `LayerOutcome`, `JTBDResult`, `Violation`, `Error`,
`Claim`, `DriftFinding`, `DriftReport`, `StyleFindings`, `JTBDPromptSet`,
`LLMVerdict`, `DriftCheck`, `ClaimPrompt`, `ClaimPromptSet`, `Verdict`,
`ReportParts`) описаны; валидируемые/сборочные типы (`AuditTarget`, `Config`,
`LLMConfig`, `JTBDPrompt`, `DriftCheck`, `DriftReport`, `ClaimPromptSet`) имеют
конструктор. `TODO: уточнить тип` нет. **[x] замкнут.**

## 9.2. Графы вызовов слайсов

Общий хвост у всех: `head -> buildReport -> (egress: sink.Write, exitCode)`.

### S1 structure
```
parseStructureArgs --Request--> runStructure
  | NewAuditTarget(Request) -> AuditTarget
  | NewConfig(Request) -> Config
  | store.ReadStructure(AuditTarget) -> RepoStructure          [I/O]
  | checkStructure(RepoStructure)[Config] -> LayerOutcome
  | buildReport(ReportParts)[AuditTarget,"structure"] -> Report
```
### S2 readability
```
  | NewAuditTarget -> AuditTarget ; NewConfig -> Config
  | store.ReadMarkdownDocs(AuditTarget) -> []MarkdownDoc        [I/O]
  | scoreReadability([]MarkdownDoc)[Config] -> LayerOutcome
  |   ← pickFormula(MarkdownDoc) ← fleschKincaid(MarkdownDoc), obornevaRus(MarkdownDoc)
  | buildReport -> Report
```
### S3 jtbd
```
  | store.ReadMarkdownDocs -> []MarkdownDoc
  | matchHeadings([]MarkdownDoc)[Config] -> HeadingIndex
  | buildJTBDCard(HeadingIndex)[spec] -> JTBDResult            (×4)
  | buildReport({JTBD:[…4]}) -> Report
```
### S4 style
```
  | linter.Run(AuditTarget) -> StyleFindings                   [I/O]
  | aggregateFindings(StyleFindings) -> LayerOutcome
  | buildReport -> Report
```
### S5 fitness
```
  | NewLLMConfig(Request) -> LLMConfig
  | store.ReadMarkdownDocs -> []MarkdownDoc
  | buildJTBDPromptSet([]MarkdownDoc)[Config] -> JTBDPromptSet
  | llm.Simulate(JTBDPromptSet) -> []LLMVerdict                [I/O]
  | scoreFitness([]LLMVerdict) -> []JTBDResult
  | buildReport -> Report
```
### S6 drift (L6a + L6c-тир за флагом, без ветвления в голове)
```
  | store.ReadStructure -> RepoStructure
  | extractClaims(RepoStructure) -> []Claim
  | NewDriftCheck(RepoStructure,[]Claim) -> DriftCheck
  | verifyClaims(DriftCheck) -> []DriftFinding                      (L6a)
  | buildClaimPromptSet(DriftCheck)[Config] -> ClaimPromptSet       (L6c-пары, cap)
  | deps.Judge.Judge(ClaimPromptSet) -> []Verdict   [I/O]          # LLMClient | NoopJudge
  | mergeSemanticFindings([]Verdict) -> []DriftFinding             (L6c)
  | NewDriftReport([]DriftFinding L6a, []DriftFinding L6c) -> DriftReport
  | buildDriftOutcome(DriftReport) -> LayerOutcome
  | buildReport -> Report
```
`--semantic` выбирает реализацию `Judge` (реальная `LLMClient` / `NoopJudge`) в
роутере — голова безусловна (skill `program-design`: нет if/циклов в трубе).
### S7 assess
```
  | layersUpTo(Request) -> LayerPlan
  | store.ReadStructure -> RepoStructure ; store.ReadMarkdownDocs -> []MarkdownDoc
  | [листья S1–S6] -> LayerOutcome (L1,L2,L3,L6a) + []JTBDResult (L4)
  | shortCircuit([]JTBDResult) -> bool
  | [L5] buildJTBDPromptSet ⨾ llm.Simulate ⨾ scoreFitness -> []JTBDResult
  | buildReport({Layers,JTBD}) -> Report
```

## 9.3. Чек-лист сверки (по всем стрелкам)

1. **Типы на стрелках существуют** в `messages.md` / stdlib — да.
2. **Имена сигнатур совпадают** с карточками — да (один словарь имён).
3. **Консеквент отправителя ⊆ антецеденту получателя** — проверено:
   - `NewAuditTarget` гарантирует читаемую директорию ⊇ предусловие `store.Read*`.
   - `store.Read*` отдаёт `RepoStructure`/`[]MarkdownDoc` ⊇ вход чистых листьев.
   - `llm.Simulate` отдаёт `[]LLMVerdict` ⊇ вход `scoreFitness`.
   - `extractClaims`+`structure` → `NewDriftCheck` → ровно вход `verifyClaims`.
4. **Типы ошибок согласованы:** все I/O возвращают sentinel из `messages.md`;
   egress (`buildErrorReport`) разбирает все восемь через `errors.Is`. Несвязанных
   с egress классов ошибок нет.
5. **Покрытие Gherkin:** каждый Then каждого `.feature` привязан к узлу графа или
   маппингу egress (таблицы `## Gherkin-mapping` в карточках). Узлов без Then нет;
   Then без узла нет.
6. **Один data-аргумент на узел:** проверено. Места слияния разнесены конструкторами:
   `NewDriftCheck` + `NewDriftReport` (S6, второй склеивает L6a+L6c-находки вместо
   ветки `if --semantic`), `JTBDPromptSet`/`ClaimPromptSet`/`ReportParts` (сборочные
   DTO). Сырых `*sql.DB`/`*http.Client`/`*exec.Cmd` в `Dependencies:`/`Deps` нет.
   Зависимость `Judge` — интерфейс (реальная/null-object), решение `--semantic` на краю.

## 9.4. Отметки

- S1 structure — **[x] согласовано**
- S2 readability — **[x] согласовано**
- S3 jtbd — **[x] согласовано**
- S4 style — **[x] согласовано**
- S5 fitness — **[x] согласовано**
- S6 drift — **[x] согласовано**
- S7 assess — **[x] согласовано**
- S8 drift --semantic — эскиз (поздний), сверяется при детализации.
