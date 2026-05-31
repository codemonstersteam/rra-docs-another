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
