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
- [x] Оператор аппрувит пакет — @maxmorev, 2026-05-27

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
- Спека: `slices/06-drift.md`. Зависимости: S1.
- Ветка `feat/slice-drift`.
- DoD: `extractClaims`/`NewDriftCheck`/`verifyClaims`/`buildDriftOutcome`; дрейф
  заявляется только при механическом подтверждении; `@wip` снят с `drift.feature`;
  зелёные.

### TICKET S7 — slice assess: CLI `assess <path>`
- Спека: `slices/07-assess.md`. Зависимости: **S1–S6 в main** (переиспользует листья).
- Ветка `feat/slice-assess`.
- DoD: `layersUpTo`; `shortCircuit`; сборка пайплайна из листьев S1–S6 (не голов);
  правило «четыре score, не усредняем»; `--up-to`; `@wip` снят с `assess.feature`;
  зелёные. **Продуктовый критерий:** `assess` на `repo-good` — четыре PASS; на
  `repo-bad` — конкретные пробелы с `file:line` ещё до LLM.

### TICKET S8 — drift --semantic (ПОЗДНИЙ, опциональный)
- Спека: `slices/08-drift-semantic.md` (эскиз). Зависимости: S6 + S5 (LLMClient).
- Детализируется отдельной итерацией program-design перед реализацией. В основной
  хендофф S1–S7 не входит.
