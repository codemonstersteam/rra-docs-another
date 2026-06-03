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

**S4 `style` (L2) — отложен в TBD.** Внешние тулзы не тянем; состав L2
проектируем отдельно от JTBD. До этого S4 не стартует.

**Следующий шаг — E16** (баг-фиксы корректности отчёта, найденные на аудите
внешнего репо `ubik-life/passkey-demo-api`). После — опциональный S8
(`drift --semantic`, L6c) или проектирование S4. Дизайн S7 —
`docs/design/assess/slices/07-assess.md`.

**Решение по приоритету E16:** дефекты #1 (commit) и #2 (ложные `broken_link`)
портят корректность отчёта на любом нормальном git-репо → **баг-фикс PR первым**,
до S8/S4. Дефекты #3 (drift строит промпты при NoopJudge) и #4 (md без `jtbd`) —
чистка/UX, отдельным PR ниже приоритетом.

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

**Статус: ⬅️ NEXT.** Первый прогон собранного тула на реальном чужом репозитории
(`assess`, `--up-to L4` + полный) показал, что пайплайн и карта отказов работают
(429 → `llm_rate_limited`/код 2 отработан точно), но вскрыл четыре дефекта.
**Решение:** два бага корректности (#1, #2) чинятся **первым PR** до любой новой
фичи; чистка (#3, #4) — отдельным PR ниже приоритетом.

**#1. `target.commit` — сырой `.git/HEAD` вместо хэша.** На обычном (не detached)
репо выдаёт `"ref: refs/heads/main\n"` вместо коммита (реальный — `f0dbd86…`).
`NewAuditTarget` не разыменовывает ref. *Severity: средняя* (поле отчёта неверное,
скоринг не ломает). **Fix:** если `.git/HEAD` начинается с `ref: ` — прочитать
указанный `.git/<ref>` (или `packed-refs`); иначе значение уже хэш.

**#2. `broken_link` — ложные блокеры на ссылках-каталогах.** L3 репортит
`[component-tests/](component-tests/)` и `[devlog/](devlog/)` битыми, хотя каталоги
существуют. Линк-чекер резолвит только файлы, не директории → ложный `fail`/код 1.
*Severity: высокая* (неверный вердикт на любом репо со ссылкой на каталог).
**Fix:** при резолве цели ссылки принимать существующий каталог как валидный
(`os.Stat` → `IsDir()` ок), не только файл.

**#3. drift строит claim-промпты при `NoopJudge`.** Дефолтный `assess` (без
`--semantic`) всё равно гоняет `buildClaimPromptSet` (561 промпт, обрезка до 20) и
сыплет WARN в stderr, хотя `NoopJudge` их игнорирует. *Severity: низкая* (лишняя
работа + шум, вывод корректен). **Fix:** пропускать построение claim-промптов,
когда judge — `NoopJudge` (L6c выключен).

**#4. `--format md` для `assess` не рендерит секцию `jtbd`.** Четыре JTBD-score
видны только в JSON; markdown показывает лишь `layers`+`violations`. *Severity:
низкая* (UX). **Fix:** добавить рендер `jtbd`-секции в markdown-вывод для `assess`/
`jtbd`/`fitness`.

**DoD первого PR (#1+#2):** оба бага закрыты, добавлены изолированные фикстуры/юниты
на разыменование ref и на ссылку-каталог; локальный CI зелёный; повторный прогон на
`passkey-demo-api --up-to L4` даёт корректный `commit` и не репортит `component-tests/`/
`devlog/` битыми.

---

## Принципы работы с backlog

- Тикет не стартует без `intent` слайса и согласованного `contracts-graph.md`.
- TBD: main всегда зелёный, ветки живут часы–день.
- Не предполагать дисциплину; работает на произвольном репо.
- При противоречии в спецификации — **остановиться и сообщить**.
