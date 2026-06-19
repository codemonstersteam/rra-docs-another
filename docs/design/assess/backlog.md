# backlog — assess (тикеты для sonnet)

Один тикет = один slice. Восходящий порядок S1→S7 (S8 поздний). Каждый тикет —
отдельная ветка/PR, `go test` + компонентные сценарии своей подкоманды зелёные,
`@wip` снимается со своего `.feature`.

## Хендофф-чеклист (заполняет opus полностью; merge PR = аппрув оператора)

- [x] Контракт (`api-specification/cli.md` + `report.schema.json`) зафиксирован, все входы слайсов описаны
- [x] Контракт содержит коды возврата 0/1/2 и `error.code` для каждого режима отказа
- [x] README содержит «Карту режимов отказа» (8 `error.code`, действие пользователя)
- [x] **Компонентные сценарии Gherkin для всех подкоманд написаны, закоммичены, стабильны (smoke зелёный; подкоманды `@wip` — happy + сценарий на каждый различимый режим отказа)**
- [x] Папка `docs/design/assess/` создана и полна
- [x] `intent.md` — задача в одну фразу
- [x] `slices.md` — таблица срезов (тип входа CLI, идентификатор, назначение)
- [x] `messages.md` — все структуры + `Result<T, Error>` (= `(T, error)` в Go)
- [x] У каждого слайса отдельная карточка с деревом модулей
- [x] У головы каждого слайса зафиксирован псевдокод пайпа (5–10 шагов)
- [x] У каждого модуля логики описаны антецедент и консеквент
- [x] У каждого I/O-модуля описан контракт и режимы отказа
- [x] **У каждого модуля Input — одна доменная структура / DTO / void; deps отдельной строкой; узлов с 2+ data-аргументами нет (слияния — через конструкторы `NewDriftCheck`, сборочные `JTBDPromptSet`/`ReportParts`)**
- [x] **I/O-зависимости инкапсулированы в `RepoStore`/`LinterRunner`/`LLMClient`/`ReportSink`; сырых `*os`/`*exec`/`*http` в контрактах нет**
- [x] **Карточка каждого слайса содержит `## Gherkin-mapping`: каждый Then привязан к узлу графа / маппингу egress**
- [x] **`contracts-graph.md` существует, граф каждого слайса согласован (`[x]`, в т.ч. покрытие Gherkin)**
- [x] Для конструкторов и чистых функций посчитаны юнит-тесты по формуле
- [x] **В таблицах юнит-тестов нет голов, I/O-модулей и ингресс-адаптеров (трубы — только компонентные сценарии)**
- [x] `infrastructure.md` — CLI-роутер + общий egress + I/O-объекты
- [x] `backlog.md` — тикеты по одному на slice, с зависимостями
- [x] Оператор аппрувит пакет — @maxmorev, 2026-06-19

## Тикеты

### TICKET S1 — slice structure: CLI `structure <path>`
- Спека: `slices/01-structure.md`. Зависимости: — (первый).
- Ветка `feat/slice-structure`.
- DoD: ингресс-адаптер; `NewAuditTarget`/`NewConfig`; `RepoStore.ReadStructure`;
  `checkStructure` + под-проверки; общий egress (`buildErrorReport`, `exitCode`,
  `ReportSink`); `buildReport`; юниты по формуле; `@wip` снят со `structure.feature`,
  happy + `path_not_found`/`read_error` зелёные; локальный CI зелёный; PR смержен.

### TICKET S2 — slice readability: CLI `readability <path>`
- Спека: `slices/02-readability.md`. Зависимости: S1 (RepoStore, egress в main).
- Ветка `feat/slice-readability`.
- DoD: `fleschKincaid`/`obornevaRus`/`pickFormula`/`scoreReadability`; L1 даёт
  только warning (никогда код 1); `buildReport`/`layerKey` копируются в
  `readability/head.go` (консолидация — на S7); `domain.Config` дополнен порогом
  читаемости; `@wip` снят со `readability.feature`; зелёные.

### TICKET S3 — slice jtbd: CLI `jtbd <path>`
- Спека: `slices/03-jtbd.md`. Зависимости: S1.
- Ветка `feat/slice-jtbd`.
- DoD: `matchHeadings` + `buildJTBDCard` ×4; четыре независимых `JTBDResult`;
  `@wip` снят с `jtbd.feature`; зелёные.

### TICKET S4 — slice style: CLI `style <path>`
- Спека: `slices/04-style.md`. Зависимости: S1.
- Ветка `feat/slice-style`.
- DoD: `LinterRunner.Run(AuditTarget)`; `aggregateFindings`; **Vale/markdownlint
  в образ раннера + Given-степы «линтеры недоступны»/«линтер падает»**; `@wip`
  снят со `style.feature` (happy + `tool_missing` + `tool_failed`); зелёные.

### TICKET S5 — slice fitness: CLI `fitness <path>`
- Спека: `slices/05-fitness.md` + ADR `docs/adr/0003-yaml-config.md`. Зависимости: S1.
- Ветка `feat/slice-fitness`.

**Реализация (Sonnet):**
- `internal/slice/fitness/`: адаптер `parseFitnessArgs` (вкл. парсинг `--llm-provider`/
  `--llm-base-url`/`--llm-model` — их в CLI ещё нет), голова `runFitness`,
  `NewLLMConfig`, `buildJTBDPromptSet`, `scoreFitness`, `register.go`.
- `internal/io/llmclient.go`: `LLMClient.Simulate(JTBDPromptSet) -> ([]LLMVerdict, error)`,
  **4 прогона внутри** (фан-аут не в голове), anthropic native + openai-совм. адаптеры,
  маппинг ошибок в `llm_rate_limited`/`llm_unavailable`/`llm_budget_exceeded`.
- **Проектный конфиг — внешний YAML** (`--config`, дефолт через `go:embed`):
  завести `gopkg.in/yaml.v3` (`go mod tidy`, первая Go-зависимость); загрузчик —
  **общая инфраструктура в `internal/cli`** (I/O на краю, не в адаптере/голове),
  битый файл → `config_invalid`; инжект value-config: `llm`→`NewLLMClient`,
  `prompts`→`Deps` слайса. Приоритет: флаг `--llm-*` > файл > вшитый дефолт.
- **Секретов в YAML нет:** `llm.api_key_env` = имя env-переменной (дефолт
  `ANTHROPIC_API_KEY`/`OPENAI_API_KEY`); ключ читается из env в `NewLLMConfig`;
  нет ключа → `ErrLLMUnavailable` **до** I/O (LLM не вызывается).
- **Дефолтные (вшитые) промпты четырёх ролей ОБЯЗАНЫ нести маркер `role:<key>`**
  (`key ∈ maintainer|consumer|manager|agent`) — контракт со стабом
  (`llm-stub` различает вердикт по роли через этот маркер). Без него режим `mixed`
  и фан-аут в четыре секции не зеленеют.

**Обвязка компонент-тестов — УЖЕ СДЕЛАНА opus (не переделывать):**
- `fitness.feature` (7 сценариев): happy `healthy` (четыре PASS + `command`/`score`/
  `gaps`), `mixed` (независимость и не-усреднение: agent FAIL/consumer PARTIAL → код 1),
  три `llm_*`-отказа с `integration "LLMClient"`, «нет ключа» → `llm_unavailable`,
  битый `--config` → `config_invalid`.
- `llm-stub` различает роль (`role:<key>`) + режимы `healthy`/`mixed`; степы
  `assertErrorCodeIntegration`/`setNoLLMKey`/`setBrokenConfig`; фикстура
  `testdata/broken-config.yml`. Стаб+степы собираются (`go vet` зелёный).

**DoD:** всё из «Реализация» сделано; `@wip` снят с `fitness.feature`; все 7
сценариев зелёные в Docker Compose; юниты по формуле (`NewLLMConfig` 3, `buildJTBDPromptSet`
2, `scoreFitness` 2); локальный CI (`vet`/`test`/`gofmt`) зелёный; PR смержен.

### TICKET S6 — slice drift: CLI `drift <path>`

**Спецификация:**
- `docs/design/assess/slices/06-drift.md` (главный документ)
- `docs/design/assess/messages.md` — `Claim`, `DriftFinding`, `DriftCheck`
- `docs/design/assess/contracts-graph.md` — секция «S6 drift»
- `docs/design/assess/infrastructure.md` — конвенция слайса, подключение в `internal/cli/cli.go`

**Зависимости:** S1 (в main) — `RepoStore`, `ReportSink`, `NewAuditTarget`, `NewConfig`, `buildReport`, egress.
Новых внешних Go-зависимостей нет.

**Ветка:** `feat/slice-drift`

**Definition of Done:**

- [x] `internal/slice/drift/domain.go`: типы `Claim{Kind,Text,File,Line}`, `DriftFinding{Claim,Reason}`, `DriftCheck`; конструктор `NewDriftCheck(structure,claims) -> DriftCheck`
- [x] `internal/slice/drift/logic.go`: `extractClaims`, `verifyClaims`, `buildClaimPromptSet`, `mergeSemanticFindings`, `NewDriftReport`, `buildDriftOutcome` — чистые функции, без I/O
- [x] `internal/io/judge.go`: интерфейс `Judge` + `NoopJudge{}` (null-object: `([], nil)`)
- [x] `internal/slice/drift/adapter.go`: `ParseArgs(args,stderr) -> (Request, error)` — парсит позиционный `[path]` и флаг `--semantic`
- [x] `internal/slice/drift/head.go`: `ProcessDrift(req, Deps) -> (Report, error)` — линейная труба по псевдокоду карточки, без ветвления
- [x] `internal/slice/drift/register.go`: `Deps{Store, Judge}` + `NewDeps(cfg, judge) -> Deps`
- [x] `internal/cli/cli.go`: `runDriftCmd` добавлен, `"drift"` убран из `subcommandsTodo`; `judge` инжектируется как `NoopJudge` (флаг `--semantic` → роутер без головы)
- [x] юнит-тесты по формуле написаны и зелёные — `go test ./...` проходит. **15 новых тестов**: `extractClaims`(2) + `NewDriftCheck`(1) + `verifyClaims`(3) + `buildClaimPromptSet`(3) + `mergeSemanticFindings`(3) + `NewDriftReport`(1) + `buildDriftOutcome`(2). Голова, адаптер, `NoopJudge` юнитами не покрываются.
- [x] компонентные тесты зелёные — `./component-tests/scripts/run-tests.sh healthy`. `@wip` снят с `drift.feature`; все три сценария зелёные: «опрятный репо → pass», «битая ссылка → fail», «путь не существует → path_not_found». Ранее зелёные сценарии S1–S5 продолжают проходить.
- [x] `backlog.md` обновлён по каждому подтверждённому пункту
- [x] `docs/design/assess/devlog.md` дополнен блоком S6
- [ ] PR создан, описание заполнено по шаблону Шага 8 скилла
- [ ] PR смержен в main, CI на main зелёный

**Ссылки на источники:**
- Скилл реализации: `skills/program-implementation/SKILL.md`
- Граф вызовов: `docs/design/assess/contracts-graph.md` S6
- Gherkin-mapping: раздел `## Gherkin-mapping` в `slices/06-drift.md`
- Принцип голова без ветвления: `slices/06-drift.md` §«Принцип: голова без ветвления»

### TICKET S7 — slice assess: CLI `assess <path>`
- Спека: `slices/07-assess.md`. Зависимости: **S1–S6 в main** (переиспользует листья).
- Ветка `feat/slice-assess`.
- DoD: `layersUpTo`; `shortCircuit`; сборка пайплайна из листьев S1–S6 (не голов);
  правило «четыре score, не усредняем»; `--up-to`; `@wip` снят с `assess.feature`;
  зелёные. **Продуктовый критерий:** `assess` на `repo-good` — четыре PASS; на
  `repo-bad` — конкретные пробелы с `file:line` ещё до LLM.

### TICKET S7-soften — смягчение гейта L5 (hasDocs + cap статикой)
- Спека: `slices/07-assess.md` (обновлена) + ADR `docs/adr/0004-soften-l5-gate.md`.
  Зависимости: **S7 в main**. Ветка `feat/slice-assess-soften-l5`.
- Суть: гейт L5 — `hasDocs(s.Docs)` вместо `!shortCircuit(L4)`; `FAIL` на L4 не
  пропускает L5, а ограничивает итог сверху до `PARTIAL` (`capL5ByL4` внутри
  `mergeOutcomes`).
- **Реализация (Sonnet):**
  - `internal/slice/assess/logic.go`: удалить `shortCircuit`; добавить
    `hasDocs(docs) bool` и приватный лист `capL5ByL4(l5, l4) map` (FAIL L4 → PASS L5
    → PARTIAL; FAIL/PARTIAL L5 и Score/Gaps не трогаем; `l4==nil` → noop); вызвать
    его в `mergeOutcomes` при формировании jtbd из L5.
  - `internal/slice/assess/head.go`: гейт `if plan.L5 && hasDocs(s.Docs)`; убрать
    `shortCircuit`. Голова без новых I/O.
  - Юниты: `hasDocs` (есть/нет docs); `capL5ByL4` (3 ветки); `mergeOutcomes` —
    ветка «L5 с кэпом». Голова не юнитится.
  - **Фикстура `component-tests/testdata/repo-soft`**: README присутствует, L1/L3/L6
    чисты (без blocker-violations), L4 проваливает роль `agent` (нет карты файлов/
    структуры под ИИ-агента), прочие роли проходят. Под сценарий «docs есть, статика
    частично провалена, ИИ годен → PARTIAL, код 0».
  - `component-tests/features/assess.feature` (обновлён): сценарий `repo-soft`
    (healthy → agent PARTIAL, код 0) зелёный; сценарий `repo-bad` переведён на режим
    стаба `bad_repo` (ИИ подтверждает провал → код 1) зелёный; happy `repo-good`
    без изменений.
- **DoD:** локальный CI зелёный (gofmt/vet/unit/component); `shortCircuit` удалён из
  кода и графа; grep-самопроверки дисциплины чисты; `backlog.md`/`devlog.md`/
  корневой `CLAUDE.md`+`backlog.md` обновлены.

### TICKET S8 — drift --semantic (ПОЗДНИЙ, опциональный)
- Спека: `slices/08-drift-semantic.md` (эскиз). Зависимости: S6 + S5 (LLMClient).
- Детализируется отдельной итерацией program-design перед реализацией. В основной
  хендофф S1–S7 не входит.
