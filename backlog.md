# Backlog — rra-docs-another

Порядок — из [`PLAN.md`](./PLAN.md): гейт (контракт + Gherkin) → проектный пакет →
семь слайсов. Правило «один тикет = один slice = одна ветка = один PR».

---

## E0. Каркас (chore)

Go-модуль, раскладка, CLI-роутер, `version`, CI на PR. **Готово:** main зелёный.

## E1. Контракт + Gherkin (гейт program-design)

`docs/intent.md` (есть), `api-specification/cli.md`, `report.schema.json`, README
«Карта режимов отказа», godog-раннер + фикстуры `repo-good`/`repo-bad`, Gherkin на
подкоманды. **Готово:** smoke зелёный, контракт зафиксирован.

## E1.1. Дотянуть обвязку компонент-тестов до полноты контракта

Раннер и smoke зелёные, но контракт `cli.md` + `report.schema.json` триггерится
не целиком. Цель — закрыть spec-уровневые пробелы единым PR. Узких фикстур
«под один слой» **не делать** — это юнит-уровень (skill `component-tests`,
секция «Граница со слоем юнитов»).

Scope:

- шаг `отчёт валидируется по api-specification/report.schema.json` + применить
  в каждом happy-сценарии;
- в happy-сценариях ассертить `command`, `schema_version`, `tool`, `target.path`;
- `git init && commit` в фикстурах (`Dockerfile.runtime`), чтобы покрыть
  `target.commit` в варианте `string`;
- сценарии под не покрытые `error.code`: `read_error` (фикстура с `chmod 000`),
  `config_invalid` (битый `--config`);
- ассертить `errors[].integration` рядом с `error.code` (контракт схемы);
- `help` / `--help` — код 0, usage в stdout;
- `--format md` для одной подкоманды (минимум: stdout — markdown, не JSON);
- `--out <файл>` против `--out -`;
- `assess --up-to L4` со `layers.L5.status="skipped"` / `L6.status="skipped"`
  (контрактная необходимость узкой фикстуры: репа, ломающая ранний слой и не
  ломающая поздний);
- вынести знание `--llm-*` из `runOnRepo` в явный степ;
- в smoke сценарии, дёргающие каждый зарегистрированный степ
  (требование skill `component-tests`, чек-лист хендоффа).

Out of scope:

- `tool_missing` / `tool_failed` — приезжают с S4 (style);
- `llm_*` — уже покрыты в `fitness.feature`;
- `--format md` для всех подкоманд (рендер — юнит-уровень).

## E2. Проектный пакет (program-design, Шаги 1–12)

`docs/design/assess/`: `slices.md`, `messages.md`, карточки слайсов,
`infrastructure.md`, `contracts-graph.md`, `backlog.md` с хендофф-чеклистом.
**Готово:** дизайн-PR смержен (= аппрув).

## E3–E9. Слайсы (program-implementation)

| Тикет | Slice | Подкоманда | Слой | Новые I/O | Статус |
|---|---|---|---|---|---|
| S1 | `structure` | `structure` | L3 | RepoStore, ReportSink | ✅ done (main) |
| S2 | `readability` | `readability` | L1 | — | ✅ done (main) |
| S3 | `jtbd-presence` | `jtbd` | L4 | — | ✅ done (main) |
| S4 | `style` | `style` | L2 | (TBD) | 🧪 **TBD — дизайн отложен** |
| S5 | `jtbd-fitness` | `fitness` | L5 | LLMClient + YAML-конфиг | ✅ done (main) |
| S6 | `drift` | `drift` | L6a | — | ✅ done (main) |
| S7 | `assess` | `assess` | L1–L6 | — | ✅ done (main, после E15) |
| S8 | `drift --semantic` | (флаг S6) | L6c | LLMClient.Judge | ⏸ follow-up, дизайн утверждён |

LLM появляется только в базовом S5 и опциональном позднем S8. S6 детерминированный
(дрейф документации), работает на любой репе.

### Где мы сейчас

Влиты **все семь основных слайсов**: S1–S3 (L3, L1, L4), S5 `fitness` (L5, LLM),
S6 `drift` (L6a) и **S7 `assess`** (полный пайплайн L1/L3/L4/L5/L6a, Option A —
одна добыча входа + листья `Evaluate`, PR #13). Предусловие **E15** (экспортный
`Evaluate` на каждый слайс, PR #12) закрыто. E12/E13/E14 закрыты.

**E16 — ✅ закрыт.** Все пять дефектов с аудита `passkey-demo-api` исправлены
(PR #15/#19/#20/#21/#22) и подтверждены прогоном: `target.commit` = реальный SHA,
L3 не репортит каталоги битыми, `doc_drift` **194 → 56** (точность L6a:
allowlist путей из конфига), drift не строит промпты при `NoopJudge`, md рендерит
`jtbd`.

**Открытой работы — две, обе опциональны:**

- **S4 `style` (L2) — отложен в TBD.** Внешние тулзы (Vale/markdownlint) не тянем;
  состав L2 проектируем отдельно и научно от JTBD. До этого S4 не стартует.
- **S8 `drift --semantic` (L6c)** — follow-up за флагом, дизайн утверждён; перед
  ним chore-промоут `LLMClient` в `internal/io`.

## E10. Эталонные фикстуры

`repo-good` / `repo-bad` + снэпшот-тесты. **Готово:** регрессии ловятся снэпшотами.

## E11. Дисциплина исходящего HTTP-I/O (skill `http-io`)

Кросс-сквозная проработка по итогам S5 (`devlog/01-llm-client-lessons.md`): все шесть
дефектов LLM-клиента закрывались в дизайне и верификации-до-кода, не в кодинге.
Вынесено в переиспользуемую дисциплину **до старта следующего слайса с HTTP-вызовом**.

Сделано:

- **Skill `skills/http-io/`** — общая дисциплина исходящего HTTP к дозируемому
  сервису: два бюджета (нагрузки и payload) считаются в дизайне слайса ДО кода;
  поток `curl-проба → машинная спека провайдера → клиент/стаб/фикстуры`; пацинг и
  бэкофф; классы отказа transient/permanent/quota; мост «от curl к тестам» (формулы
  юнитим, контрактные ветки — компонентом). `skills/llm-client/` ужат до
  LLM-специфики (протокол, `response_format`, фан-аут ролей) и ссылается на `http-io`.
- **Спека провайдера** — `api-specification/providers/anthropic-openai-compat.openapi.yaml`
  (OpenAPI 3.1 на проверенный curl-контракт; источник истины для клиента, стаба, фикстур).
- **Рефактор кода S5 (поведение сохранено, 21/21 компонент-сценариев зелёные):**
  - чистые функции-формулы в `fitness/logic.go` (`estimateTokens`, `promptSetTokens`,
    `overTokenBudget`, `retryWait`) + юниты;
  - пре-флайт payload-бюджета в `LLMClient.Ask` (не слать заведомо лишний контекст);
  - защитный лимит токенов вынесен в конфиг (`llm.token_budget`, дефолт 300000);
  - бэкофф по `Retry-After` на 429 (`llm.max_retries`, дефолт 0 = прежнее поведение).

**Правило для S6/S7/S8 и любого нового HTTP-I/O:** дизайн-карта слайса проходит
чеклист дизайна из `http-io` (бюджеты посчитаны, спека провайдера есть, режимы
стаба из контракта) — ДО кода.

## E12. Техдолг — изолированные фикстуры компонентных тестов

**Статус: ✅ закрыто.** Созданы 10 изолированных фикстур
(`repo-good-<slice>` / `repo-bad-<slice>` для structure/readability/jtbd/fitness/drift);
`.feature` каждого слайса переведён на свою пару; `repo-good`/`repo-bad` оставлены
под `assess` (S7). Компонентные тесты 24/24 зелёные. Текст ниже — история.

Скилл `component-tests` (обновлён по итогам S6) требует: каждый слайс имеет
**собственную пару фикстур** (`repo-good-<slice>`, `repo-bad-<slice>`), не разделяемую
с другими слайсами. Сейчас все пять реализованных подкоманд (S1–S3, S5, S6) тестируются
против одной пары `repo-good` / `repo-bad` — нарушение изолированной модели.

Симптом, который это вскрыл: при реализации S6 `drift` `repo-good` оказалась
несамосогласованной с новым check-типом (AGENTS.md ссылался на несуществующие пути),
что потребовало ручного аудита и правки фикстуры в ходе реализации слайса, а не до.

**Scope:**
- Создать `repo-good-structure`, `repo-bad-structure` (минимальные, только для L3).
- Создать `repo-good-readability`, `repo-bad-readability` (только для L1).
- Создать `repo-good-jtbd`, `repo-bad-jtbd` (только для L4).
- Создать `repo-good-fitness`, `repo-bad-fitness` (только для L5).
- Создать `repo-good-drift`, `repo-bad-drift` (только для L6a).
- Обновить степы и `.feature`-файлы, чтобы каждый сценарий ссылался на свою фикстуру.
- `repo-good` / `repo-bad` оставить как фикстуры для будущего `assess` (S7).

**Приоритет:** был «до реализации S7». Закрыто — фикстуры изолированы до старта S7.

## E13. Техдолг — целостность скилла `program-implementation`: определить «локальный CI»

**Статус: ✅ закрыто.** Уже реализовано в `skills/program-implementation/SKILL.md`:
Шаг 4 определяет «локальный CI» как четыре шага, зеркалящие `ci.yml`
(`gofmt -l .` → `go vet ./...` → `go test ./...` → `run-tests.sh`); gofmt/vet входят
в критерий зелёного и в сводку оператору (Шаг 7.3). Backlog-пункт ниже — история.

**Проблема (история).** Скилл использует термин «локальный CI зелёный» (Шаги 4, 6, 7, 8, DoD
тикетов), но нигде не определяет его состав. В результате Sonnet раз за разом прогоняет
только `go test ./...` + компонентные тесты и пропускает `gofmt -l .` и `go vet ./...`,
которые реально гонит CI на PR (`.github/workflows/ci.yml`). Ошибка воспроизводится
несмотря на запись в памяти — потому что память не читается так же надёжно, как скилл.

**Три места несогласованности:**

1. **Шаг 4** — список команд не включает `gofmt` и `go vet`.
2. **«Локальный CI» не определён** — термин без состава, каждый интерпретирует по-своему.
3. **Шаг 7.3 (сводка оператору)** — шаблон не содержит строки для `gofmt`/`vet`.

**Scope:**
- В Шаге 4 добавить `gofmt -l .` и `go vet ./...` перед `go test ./...` и явно
  определить «локальный CI» как четырёхшаговый чеклист, зеркалящий `ci.yml`.
- В Шаге 7.3 добавить в шаблон сводки строку `- gofmt/vet: чисто`.
- Проверить остальные шаги на упоминание «CI» и убедиться, что везде
  подразумевается один и тот же определённый состав.

**Приоритет:** был «до старта S7». Закрыто — определение «локального CI» уже в скилле.

## E14. Конформанс с ADR 0003 — хардкод проверок → конфиг

**Статус: ✅ закрыто (PR #9).** Часть проверок жила хардкодом в Go вопреки
ADR 0003 («словари L4 в YAML», проектный конфиг — источник истины). Вынесено
в дефолтный `internal/domain/defaults/config.yaml` через узкие доменные
value-object'ы; голова слайса достаёт срез из `Config`, не таская весь `Config`
в чистую логику.

**Сделано:**

- **L4 (jtbd):** словари обязательных секций по ролям и карта ролей →
  `jtbd.consumers` (`name`/`synonyms`/`critical`); типы `JTBDSpec`/`JTBDConsumer`/
  `JTBDSection` + `Config.JTBDSpec()` с валидацией. Кастомный `--config` без
  `jtbd` → `config_invalid` (решение оператора, не тихий PASS). Роли больше не
  фиксированы в Go.
- **L3 (structure):** обязательные файлы → `required_files` (дефолт `[README.md]`);
  код нарушения `missing_readme` → generic `missing_required_file`.
- **L6 (io):** список известных манифестов → `manifests` (прежние 5); `ReadStructure`
  принимает список из `cfg.Manifests()`, ничего не хардкодит.

Поведение по умолчанию (встроенный конфиг) сохранено — контракт не менялся,
компонентные тесты зелёные.

## E15. Рефактор — экспортный `Evaluate` на слайс (предусловие S7)

**Статус: ✅ закрыто (PR #12).** S7 `assess` собирает аудит из листьев S1–S6 за
**один** проход. Чтобы не дублировать добычу входа (`NewAuditTarget` + `NewConfig` +
чтение репы) на каждый слой, каждый слайс выставляет один экспортный вход «оценка
слоя поверх уже прочитанных данных», а его голова делегирует туда же. Это Option A
из обсуждения дизайна S7 (Option B — «assess зовёт пять голов» — отклонён: давал
5× валидацию и 5× чтение ФС на каждом прогоне; см. `07-assess.md`).

**Ключ:** `RepoStore.ReadStructure` уже возвращает `RepoStructure{Files, Docs,
Manifests, MTimes}`, где `Docs` = те же `[]MarkdownDoc`, что отдаёт
`ReadMarkdownDocs`. Значит одного чтения хватает всем пяти слоям → в `assess`
валидация и чтение по **1×**.

**Scope (слайсы S1–S6; поведение голов и отчётов неизменно):**

- `structure.Evaluate(s RepoStructure, cfg Config) LayerOutcome` (= `checkStructure`).
- `readability.Evaluate(docs []MarkdownDoc, cfg Config) LayerOutcome` (= `scoreReadability`).
- `jtbd.Evaluate(docs []MarkdownDoc, cfg Config) map[string]JTBDResult`
  (`matchHeadings` + `buildJTBDCard`×N).
- `fitness.Evaluate(docs []MarkdownDoc, cfg Config, llm LLMClient) (map[string]JTBDResult, error)`
  (`buildJTBDPromptSet` → `Ask` → `scoreFitness`). Фильтр `cfg.Docs()` становится
  **in-memory** по `docs` (заменяет отдельное чтение `ReadMarkdownDocsByList`; голова
  fitness сводится к одному `ReadMarkdownDocs` + фильтр в `Evaluate`).
- `drift.Evaluate(s RepoStructure, cfg Config, judge Judge) (LayerOutcome, error)`
  (`extractClaims` → `verifyClaims` → `buildClaimPromptSet` → `judge` →
  `mergeSemanticFindings` → `buildDriftOutcome`).
- Голова каждого слайса: acquire (`NewAuditTarget` + `NewConfig` + `Read`) →
  `Evaluate` → `buildReport`. Сигнатуры голов/`Deps` снаружи не меняются.

**DoD:**

- Каждый `Evaluate` покрыт юнит-тестами по формуле (happy + ветки); новая логика
  (in-memory фильтр `cfg.Docs()` в `fitness.Evaluate`) — отдельными ветками.
- Голова делегирует в `Evaluate`; поведение не меняется → компонентные тесты всех
  слайсов зелёные **без правок** `.feature`.
- Локальный CI зелёный (`gofmt -l .` → `go vet ./...` → `go test ./...` →
  `run-tests.sh`).

**Порядок:** отдельным PR **перед** реализацией `07-assess.md`.

## E16. Дефекты, найденные на аудите внешнего репо (`ubik-life/passkey-demo-api`)

**Статус: ✅ закрыт.** Первый прогон собранного тула на реальном чужом репозитории
(`assess`, `--up-to L4` + полный) показал, что пайплайн и карта отказов работают
(429 → `llm_rate_limited`/код 2 отработан точно), но вскрыл пять дефектов — все
исправлены: #1, #2 (PR #15); #5 — T1 (PR #19) + T1b (PR #20) + T1c (PR #22);
#3, #4 — T2/T3 (PR #21). `doc_drift` на `passkey-demo-api`: **194 → 152 → 68 → 56**;
остаточные ложные классы (OpenAPI `paths.*`, Go-символы `pkg.Symbol`, `~/`-абсолюты)
устранены — allowlist расширений вынесен в конфиг (`link_extensions`, ADR 0003).

**#1. ✅ done (PR #15). `target.commit` — сырой `.git/HEAD` вместо хэша.** На обычном
(не detached) репо выдавал `"ref: refs/heads/main\n"` вместо коммита. `NewAuditTarget`
не разыменовывал ref. **Сделано:** `ref: ` разыменовывается в хэш.

**#2. ✅ done (PR #15). `broken_link` — ложные блокеры на ссылках-каталогах.** L3
репортил `[component-tests/](component-tests/)` и `[devlog/](devlog/)` битыми, хотя
каталоги существуют. **Сделано:** существующий каталог принимается как валидная цель
ссылки (`IsDir()`), не только файл.

**#3. drift строит claim-промпты при `NoopJudge`.** Дефолтный `assess` (без
`--semantic`) всё равно гоняет `buildClaimPromptSet` (561 промпт, обрезка до 20) и
сыплет WARN в stderr, хотя `NoopJudge` их игнорирует. *Severity: низкая* (лишняя
работа + шум, вывод корректен). **Fix:** пропускать построение claim-промптов,
когда judge — `NoopJudge` (L6c выключен).

**#4. `--format md` для `assess` не рендерит секцию `jtbd`.** Четыре JTBD-score
видны только в JSON; markdown показывает лишь `layers`+`violations`. *Severity:
низкая* (UX). **Fix:** добавить рендер `jtbd`-секции в markdown-вывод для `assess`/
`jtbd`/`fitness`.

**#5. L6a `extractClaims` — низкая точность извлечения путей (много ложных
`doc_drift`).** *Severity: средняя→высокая* (тот же класс, что #2: ложные
blocker'ы → неверный `fail`/код 1). На `passkey-demo-api` L6 = `fail`, score 0,
**194 `doc_drift`** — смесь реальных (`docs/architecture.md`, `docs/adr` упомянуты,
но отсутствуют — подтвердил и LLM в gaps) и **ложных срабатываний**.

Корень — предикат `isFilePath` в `internal/slice/drift/logic.go` (≈ строка 72).
Сейчас он отбраковывает только: пусто, без `/`, есть `://`, префикс `-`, символы
`* ? < > пробел`. **Пропускает** (→ ложные claim-kind `link`):

| Класс ложного «пути» | Пример из прогона | Почему путь не должен матчиться |
|---|---|---|
| git SSH/scp-URL | `git@github.com:ubik-life/passkey-demo-ui.git` | содержит `@`; это git-remote, не файл |
| API-роут / плейсхолдер | `/v1/...` | `...`-эллипсис + версионный сегмент роута |
| внешняя org/repo-ссылка | `ubik-life/concept` | не локальный путь репо |
| абсолютный путь-концепт | `/migrations`, `/contract-tests` | ведущий `/` (filesystem-абсолют, в репо не резолвится) |

**Fix (расширить `isFilePath`, высокая точность — не отбросить реальные пути вроде
`docs/adr`, `docs/architecture.md`):** дополнительно исключать строки, которые
- содержат `@` (git-remote / e-mail);
- содержат `...` (плейсхолдер);
- начинаются с `/` (filesystem-абсолют — claim резолвится относительно doc/корня,
  не от корня ФС; ведущий `/` = почти всегда роут/концепт);
- похожи на host (`<segment>.<tld>/…`, напр. `github.com/...`) — опционально, по
  усмотрению при реализации.

Вторично (отметить, не обязательно в этом PR): claim-kind `dependency` матчит
`strings.Contains(line, pkg)` по всей строке — тоже грубо; вынести в отдельный пункт
если всплывут ложные `dependency`-находки.

**DoD #5:** `isFilePath` расширен; юнит-тесты по формуле на каждый новый класс-ветку
(git-URL, `...`, ведущий `/`, плюс happy: `docs/adr`, `docs/architecture.md`,
`internal/io/repostore.go` остаются валидными путями); локальный CI зелёный;
повторный прогон `assess /tmp/passkey-demo-api`: число `doc_drift` падает за счёт
ложных, реальные (`docs/architecture.md`, `docs/adr`) сохраняются.

### Тикеты E16 (декомпозиция для Sonnet)

Контракт (`report.schema.json`, `cli.md`) не меняется — это баг-фиксы поведения.
План PR: **T1 первым** (корректность), затем T2+T3 (можно одним cleanup-PR).

#### E16-T1 — fix: точность `extractClaims` (#5) · ✅ done (PR #19), частично

**Сделано:** `isFilePath` отбрасывает `@`, `...`, ведущий `/`. Прогон на
`passkey-demo-api`: `doc_drift` **194 → 152**, явные ложные пути ушли. **Остаток** —
не-пути с `/`, что эвристика «contains `/`» всё ещё пропускает (Go import-path,
MIME, имена веток, внешние `org/repo`): вынесен в **E16-T1b**.

**Спецификация:** E16 #5 (выше). **Ветка:** `fix/extractclaims-precision`.

- [ ] `internal/slice/drift/logic.go` — расширить чистый предикат `isFilePath`:
  дополнительно возвращать `false`, если строка содержит `@` (git-remote/e-mail),
  содержит `...` (плейсхолдер), начинается с `/` (filesystem-абсолют). Опционально:
  host-подобные `<seg>.<tld>/…`. Реальные пути (`docs/adr`, `docs/architecture.md`,
  `internal/io/repostore.go`) остаются валидными.
- [ ] `internal/slice/drift/logic_test.go` — юниты по формуле: 1 happy + ветки на
  каждый новый reject-класс (`git@github.com:ubik-life/x.git`, `/v1/...`,
  `/migrations`) + позитивные кейсы реальных путей (не должны отбрасываться).
- [ ] компонентные drift-тесты зелёные **без правок** `.feature` (предикат —
  юнит-уровень, не новая контракт-ветка → новый сценарий не нужен; правило
  различимости `skills/component-tests`).
- [ ] локальный CI зелёный (`gofmt -l .` → `go vet ./...` → `go test ./...` →
  `run-tests.sh`).
- [ ] ручная верификация: `assess <путь-к-passkey-demo-api>` — `doc_drift` падает
  за счёт ложных (`git@…`, `/v1/...`, `ubik-life/concept` уходят), реальные
  (`docs/architecture.md`, `docs/adr`) сохраняются.

#### E16-T1b — ✅ done (PR #20). fix: якорь claim-путей к структуре репо (остаток #5)

**Проблема (техдолг).** После T1 эвристика `isFilePath` всё ещё чисто
синтаксическая: «есть `/`, нет `@`/`...`/ведущего `/`». Под неё на
`passkey-demo-api` попадают **152 `doc_drift`**, из которых заметная доля — НЕ
файловые пути (counts с прогона `drift`):

| Класс ложного «пути» | Пример | ≈counts |
|---|---|---|
| Go import-path | `github.com/golang-jwt/jwt/v5`, `github.com/descope/virtualwebauthn` | 11 |
| Go stdlib/пакет | `crypto/rand` | 4 |
| Go-модуль (short) | `mattn/go-sqlite3` | 4 |
| MIME-тип | `application/json` | 4 |
| имя ветки | `refactor/s1-s2-store` | 7 |
| внешний org/repo | `ubik-life/concept`, `codemonstersteam/…/component-tests` | 5 |

Точечными reject-правилами это не добить (классов много, формы пересекаются с
настоящими путями). Нужен **сдвиг подхода**: claim-путь признаётся, только если он
**похож на путь внутрь этого репо**, а не на произвольный токен со слэшем.

**Решение (концретно, для реализации — Sonnet не проектирует, реализует).**
Признавать `link`-claim, если выполнено **хотя бы одно**:
1. у токена есть **расширение файла** (последний сегмент содержит `.<ext>`, где
   `<ext>` — буквы/цифры 1–8 символов), **или**
2. **первый сегмент** токена совпадает с реальным **топ-уровневым элементом** репо
   (каталог или файл в корне), вычисленным из `structure.Files`.

Иначе — не claim. Проверка на каждом наблюдённом классе (должно дать ожидаемое):

| Токен | ext? | top-seg? | вердикт |
|---|---|---|---|
| `docs/architecture.md` | `.md` | `docs`✓ | **claim** (реальный дрейф) |
| `scripts/run-tests.sh` | `.sh` | — | **claim** |
| `docs/adr`, `internal/auth`, `devlog/06` | — | ✓ | **claim** |
| `github.com/...`, `crypto/rand`, `mattn/go-sqlite3` | — | — | reject |
| `application/json`, `refactor/s1-s2-store`, `ubik-life/concept` | — | — | reject |

Топ-уровневый набор = `{ первый сегмент f | f ∈ structure.Files }` (даёт и корневые
файлы, и каталоги первого уровня). Извлечение получает его из `structure` —
данные уже на руках; чистую функцию-предикат параметризовать этим набором.

**Спецификация:** E16 #5 + это тело. **Ветка:** `fix/extractclaims-precision-v2`.

**DoD:**

- [ ] `internal/slice/drift/logic.go` — заменить/дополнить синтаксический
  `isFilePath` предикатом «похоже на путь внутрь репо» (ext **или** top-seg);
  топ-набор строится один раз в `extractClaims` из `structure.Files` и передаётся
  в предикат (чистая функция, без I/O). Правила T1 (`@`/`...`/ведущий `/`)
  сохранить.
- [ ] `internal/slice/drift/logic_test.go` — юниты по формуле: happy (`docs/adr`
  при top-seg `docs`; `x/y.md` по расширению) + ветки на каждый reject-класс
  (`github.com/a/b`, `crypto/rand`, `application/json`, `refactor/x`,
  `ubik-life/concept` без top-seg) + граница (top-seg есть, расширения нет → claim).
- [ ] компонентные drift-тесты зелёные **без правок** `.feature` (предикат —
  юнит-уровень). Если изолированной фикстуре `repo-*-drift` не хватает кейса для
  регрессии — добавить **в неё** (не новый сценарий), по правилу различимости.
- [ ] локальный CI зелёный (`gofmt -l .` → `go vet ./...` → `go test ./...` →
  `run-tests.sh`).
- [ ] ручная верификация: `drift <путь-к-passkey-demo-api>` — `doc_drift` резко
  падает (уходят import-path/MIME/ветки/внешние repo), реальные (`docs/architecture.md`,
  `docs/adr`, `scripts/run-tests.sh`, `internal/auth`) сохраняются.

#### E16-T1c — ✅ done (PR #22). fix: расширение по allowlist из конфига (остаток #5)

**Проблема (техдолг).** После T1b (`doc_drift` 194→68) ветка предиката «есть
расширение файла» принимает **любой** `.<1–8 alnum>`. На полном прогоне
`assess`/`drift` по `passkey-demo-api` через это всё ещё ложно матчатся:

| Класс | Пример | ≈ |
|---|---|---|
| OpenAPI path-pointer | `paths./sessions/{id}/assertion.post`, `paths./users/me.get` | 6 |
| Go-символ `pkg.Symbol` | `crypto/rand.Reader`, `crypto/rand.Read` | 4 |
| home/внешний абсолют | `~/IdeaProjects/web-book/…md` | 2 |

`.post`, `.get`, `.Reader`, `.Read` — не расширения файлов; ведущий `~` — абсолют
не в репо.

**Решение (для реализации — Sonnet не проектирует, реализует).**
1. Заменить «есть расширение = `.<1–8 alnum>`» на **allowlist реальных расширений**
   (последний сегмент оканчивается на `.<ext>`, `<ext>` ∈ списке, регистронезависимо).
   **Список — из конфига, не хардкод** (ADR 0003 / E14: словари в YAML, как
   `manifests`/`required_files`/`jtbd`). Новая секция в `internal/domain/defaults/
   config.yaml` — `link_extensions:` со стартовым набором: `md markdown txt go mod
   sum sh bash yml yaml json toml feature sql proto py rs ts js tsx html css`.
   Доступ — `Config.LinkExtensions() []string` (зеркало `Config.Manifests()`).
   Валидация по образцу `manifests`: секция **обязательна** — кастомный `--config`
   без `link_extensions` → `config_invalid` (решение оператора, не тихий дефолт).
2. Дополнить reject-правила T1: отбрасывать токены с ведущим `~` (как уже `/`).
3. Ветку «первый сегмент = топ-уровневый элемент репо» (из T1b) **сохранить** — она
   держит реальные бесрасширенные ссылки (`docs/adr`, `internal/auth`).

Предикат остаётся чистой функцией: allowlist приходит параметром (голова/`Evaluate`
достаёт `cfg.LinkExtensions()` и передаёт вниз — как `cfg.Manifests()` в `extractClaims`).

Проверка по классам: `paths./users/me.get` (ext `get`∉list, top-seg `paths.`∉repo →
reject), `crypto/rand.Reader` (ext `Reader`∉list, top-seg ∉repo → reject),
`~/…md` (ведущий `~` → reject); happy сохраняются: `docs/architecture.md`(`md`),
`scripts/run-tests.sh`(`sh`), `features/smoke.feature`(`feature`), `docs/adr`(top-seg).

> Остаток `web-book/docs/…md` (внешний repo с реальным `.md` и без repo-top-seg)
> по синтаксису не отличим от `scripts/run-tests.sh` — оставляем (1–2 находки),
> ловить семантикой не оправдано.

**Спецификация:** E16 #5 + это тело. **Ветка:** `fix/extractclaims-precision-v3`.

**DoD:**

- [ ] `internal/domain/defaults/config.yaml` — секция `link_extensions:` со
  стартовым набором (комментарий: секция обязательна, как `manifests`).
- [ ] `internal/domain/domain.go` — `Config.LinkExtensions() []string` + валидация
  (отсутствие в кастомном `--config` → `ErrConfigInvalid`), по образцу `Manifests()`.
- [ ] `internal/domain/domain_test.go` — юнит на `LinkExtensions()` + ветка
  `config_invalid` без секции (по образцу существующих тестов конфига).
- [ ] `internal/slice/drift/logic.go` — extension-ветка предиката проверяет allowlist
  **из параметра** (`cfg.LinkExtensions()`, прокинут через `extractClaims`); добавлен
  reject ведущего `~`; ветка top-seg из T1b сохранена; всё — чистые функции, без I/O.
- [ ] `internal/slice/drift/logic_test.go` — юниты по формуле: ветки на reject
  (`paths./users/me.get`, `crypto/rand.Reader`, `~/x.md`) + happy (`docs/architecture.md`,
  `scripts/run-tests.sh`, `x.feature`, `docs/adr` по top-seg).
- [ ] компонентные drift-тесты зелёные **без правок** `.feature` (предикат —
  юнит-уровень; при нехватке регрессии — кейс **в** `repo-*-drift`).
- [ ] локальный CI зелёный (`gofmt -l .` → `go vet ./...` → `go test ./...` →
  `run-tests.sh`).
- [ ] ручная верификация: `drift <путь-к-passkey-demo-api>` — `doc_drift` падает с 68
  (уходят `paths.*`, `crypto/rand.*`, `~/…`), реальные пути сохраняются.

#### E16-T2 — ✅ done (PR #21). chore: drift не строит claim-промпты при `NoopJudge` (#3)

**Спецификация:** E16 #3. **Ветка:** `chore/drift-skip-noop-judge`.

- [ ] `internal/io/judge.go` — добавить в интерфейс `Judge` метод `Enabled() bool`;
  `NoopJudge.Enabled()` → `false` (null-object). (Альтернатива: type-assert
  `NoopJudge` в `Evaluate` — допустима, но `Enabled()` чище.)
- [ ] `internal/slice/drift/evaluate.go` — если `!judge.Enabled()`: пропустить
  `buildClaimPromptSet` и `judge.Judge` (semantic-находки пусты, как и сейчас).
- [ ] WARN `buildClaimPromptSet: обрезка до max_judge_calls` больше не появляется
  на дефолтном `assess`/`drift` (без `--semantic`) — проверить прогоном.
- [ ] вывод не меняется (с `NoopJudge` semantic и так был пуст) → компонентные
  drift-тесты зелёные без правок; локальный CI зелёный.

#### E16-T3 — ✅ done (PR #21). fix: `--format md` рендерит секцию `jtbd` (#4)

**Спецификация:** E16 #4. **Ветка:** `fix/md-render-jtbd`.

- [ ] `internal/io/reportsink.go` — в `renderMarkdown` добавить секцию `## JTBD`:
  по каждому потребителю (`maintainer`/`consumer`/`manager`/`agent`) — `status`,
  `score`, список `gaps`. Опц.: стабилизировать порядок `Layers`/`JTBD`
  (сейчас итерация по map недетерминирована).
- [ ] юнит на `renderMarkdown` (чистая функция форматирования, юнит-уровень — см.
  E1.1): отчёт с заполненным `JTBD` → markdown содержит секцию и четыре потребителя.
- [ ] ручная проверка: `assess --format md` показывает четыре JTBD-score; локальный
  CI зелёный.

---

## Принципы работы с backlog

- Тикет не стартует без `intent` слайса и согласованного `contracts-graph.md`.
- TBD: main всегда зелёный, ветки живут часы–день.
- Не предполагать дисциплину; работает на произвольном репо.
- При противоречии в спецификации — **остановиться и сообщить**.
