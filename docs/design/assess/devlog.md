# devlog — assess

## S6 — drift: CLI `drift <path>` (2026-05-31)

**Что сделано:** Реализован слайс S6 `drift` (L6a, детерминированный, без LLM):
`extractClaims` (link/dependency из backtick-inline), `verifyClaims` (резолв против
Files/Manifests), `buildClaimPromptSet`/`mergeSemanticFindings`/`NewDriftReport`
(L6c-заглушки через NoopJudge), `buildDriftOutcome`. Интерфейс `Judge` + `NoopJudge`
в `internal/io`. Подключено в `internal/cli/cli.go`.

**Решения, принятые по ходу:**
- Резолвинг путей — двойной: сначала от корня репо (наиболее частое соглашение
  в документации), затем относительно doc-файла. Это позволяет писать `cmd/api`
  в `docs/architecture.md` не указывая `../`.
- Директория считается существующей, если в `Files` есть хотя бы один файл
  с таким префиксом (иначе `cmd/api` как dir-reference не резолвится).
- Шаблоны вида `` `feat/<name>` `` не считаются путями (фильтр `<>`).
- Фикстура `repo-good` дополнена: добавлены `cmd/api/main.go`,
  `internal/plan/cut.go`, `internal/store/store.go` — файлы, на которые
  ссылается AGENTS.md виджет-сервиса.

**Тесты:** юниты 15, coverage 100% по строкам логики. Компонентные сценарии:
«опрятный репо → pass», «битая ссылка → fail», «путь не существует →
path_not_found» — все зелёные. Ранее зелёные S1–S5 не регрессировали (24/24).

## S7-soften — смягчение гейта L5 (2026-06-19)

**Что сделано:** Гейт L5 переведён с `!shortCircuit(L4)` на `hasDocs(s.Docs)` —
ИИ-проверка пригодности запускается при наличии документации, а не только когда
вся статика L4 прошла. `shortCircuit` удалён; добавлены `hasDocs` и приватный лист
`capL5ByL4` (внутри `mergeOutcomes`): FAIL на L4 ограничивает PASS L5 до PARTIAL.
Цель — прогон на реальных проектах и сбор статистики L4/L5 (ADR 0004).

**Решения, принятые по ходу:**
- Сигнатуру `mergeOutcomes(plan, target, out)` НЕ переставляли под карточку
  (`out` уже единственный data-агрегатор `layerOutcomes`); порядок аргументов —
  косметика, перестановка сломала бы 3 рабочих теста зря.
- Капинг двух карт `JTBDResult` оставлен внутри `mergeOutcomes` (один data-аргумент
  через агрегатор), `capL5ByL4` — приватный лист, не узел графа: соблюдено правило
  одного аргумента.
- Фикстура `repo-soft`: README (manager «умеет» + consumer «запуск»/«api») +
  `docs/architecture.md` («Архитектура») + `CONTRIBUTING.md` («Contributing») дают
  maintainer/consumer/manager PASS; секции agent (`agents`/`агент`/`контекст`) нет —
  L4 валит только `agent`. L1 — warning (низкая читаемость FRE 51, не блокер),
  L3/L6 чисты. С healthy-стабом L5 даёт agent PASS → `capL5ByL4` → PARTIAL → код 0.

**Тесты:** юниты +6 (`hasDocs`×2, `capL5ByL4`×3, `mergeOutcomes` cap-ветка×1),
−3 (`shortCircuit`); `go test ./...` зелёный. Компонентные сценарии assess:
«опрятный → 4×PASS код 0», «docs есть, статика частично провалена → agent PARTIAL
код 0», «битый `bad_repo` → код 1», «путь не существует → код 2» — все зелёные.
Локальный CI 4/4 (gofmt/vet/unit/component).
