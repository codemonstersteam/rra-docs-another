---
name: llm-client
description: Проектирование и реализация модуля взаимодействия с LLM-провайдером (Anthropic, OpenAI-совместимые). Применять, когда нужно добавить LLM-вызов в CLI-тул или сервис — выбрать протокол, спроектировать I/O-объект, настроить structured output, обработать режимы отказа, написать стаб для компонентных тестов. Не применять для выбора модели под задачу — это отдельное решение.
---

# LLM-клиент — проектирование модуля взаимодействия

Скилл собирает уроки реальной реализации `LLMClient` для Anthropic/OpenAI
в CLI-туле (`internal/slice/fitness/io.go`). Каждый раздел — зафиксированное
решение с причиной, а не рекомендация.

---

## Протокол: OpenAI-совместимый слой, не нативный API

Используем **OpenAI chat completions format** (`POST /v1/chat/completions`)
даже для Anthropic. Причина: один код работает с любым провайдером —
Anthropic, OpenAI, локальный Ollama, корпоративный прокси.

```
Anthropic base URL: https://api.anthropic.com/v1
Endpoint:          POST {base_url}/chat/completions
Auth header:       Authorization: Bearer <key>
```

Нативный Anthropic API (`POST /v1/messages`, заголовок `x-api-key`,
поле `anthropic-version`) — **не используем**. Он даёт больше возможностей
(thinking, citations), но ломает провайдер-агностичность.

**Ловушка:** дефолтный `baseURL = "https://api.anthropic.com"` без `/v1`
даёт `POST https://api.anthropic.com/chat/completions` — 404 или 400.
Всегда включать `/v1` в `baseURL`.

---

## Structured output — обязателен

Без `response_format` модель оборачивает JSON в markdown-блок
(` ```json ... ``` `), даже при явной инструкции «только JSON».
Это ломает парсинг.

**Правильный запрос для Anthropic (OpenAI-слой):**

```json
{
  "model": "claude-sonnet-4-6",
  "messages": [...],
  "max_tokens": 4096,
  "response_format": {
    "type": "json_schema",
    "json_schema": {
      "name": "verdict",
      "strict": true,
      "schema": {
        "type": "object",
        "properties": {
          "status": {"type": "string", "enum": ["PASS", "FAIL", "PARTIAL"]},
          "score":  {"type": "integer"},
          "gaps":   {"type": "array", "items": {"type": "string"}}
        },
        "required": ["status", "score", "gaps"],
        "additionalProperties": false
      }
    }
  }
}
```

**Ловушки `response_format` у Anthropic:**
- `"type": "json_object"` → 400 `Input should be 'json_schema'`
- Без `"strict": true` → 400 `Field required`
- Без `"additionalProperties": false` → 400 при некоторых схемах

С `json_schema + strict: true + additionalProperties: false` модель возвращает
чистый JSON строкой в `choices[0].message.content` без обёрток.

**Fallback-парсинг** оставляем как второй эшелон защиты — на случай
провайдера, не поддерживающего `response_format`:

```go
// extractJSON вырезает первый JSON-объект из строки ответа модели,
// игнорируя markdown-обёртки и любой текст вокруг.
func extractJSON(s string) string {
    start := strings.Index(s, "{")
    end := strings.LastIndex(s, "}")
    if start == -1 || end == -1 || end < start {
        return s
    }
    return s[start : end+1]
}
```

---

## Конфигурация: секрет через env, не в YAML

YAML-конфиг хранит **имя** переменной окружения, не сам ключ:

```yaml
llm:
  provider: anthropic
  model: claude-sonnet-4-6
  api_key_env: ANTHROPIC_API_KEY   # имя переменной, не значение
  base_url: ""
```

Ключ читается из `os.Getenv(cfg.APIKeyEnv)` при каждом вызове `Simulate`.
Так конфиг можно коммитить. Если переменная не задана — fail-fast с
`ErrLLMUnavailable` до любого I/O.

---

## Управление токенами

**Бюджет на вызов** (`max_tokens`) — в YAML-конфиге или конст в коде,
не хардкод. Для задач оценки документации: 4096 достаточно на ответ.

**Защитный лимит** (`tokenBudgetLimit`) проверяется по `usage.total_tokens`
из ответа. Значение должно покрывать реальные репозитории:
- Маленькая репа (3-4 doc-файла): ~5k–20k токенов за вызов
- Средняя репа (passkey-demo-api, 4 файла): ~15k–30k токенов
- При отправке всех `.md` подряд: 200k+ токенов → rate limit и медленно

**Фильтр документов обязателен.** Отправлять только нужные файлы,
список в конфиге:

```yaml
docs:
  - README.md
  - CLAUDE.md
  - CONTRIBUTING.md
  - AGENTS.md
```

`ReadMarkdownDocsByList` читает только их, пропускает отсутствующие.
Без фильтра репа с крупными docs (851KB) съедает весь TPM-лимит
на первом же вызове из четырёх.

---

## Rate limiting: пауза между вызовами

При N последовательных вызовах (4 JTBD-роли) каждый отправляет один
и тот же корпус документов. Если первый вызов уходит успешно,
второй может упасть с 429, если суммарный TPM превышен.

Добавить задержку между вызовами:

```go
for i, p := range set.Prompts() {
    if i > 0 {
        time.Sleep(10 * time.Second)
    }
    v, err := c.call(p, key)
    ...
}
```

10 секунд — рабочий интервал для Anthropic tier-1 при ~15k токенов/вызов.
При больших объёмах или низком тире — увеличить.

---

## Доменные ошибки

Три различимых режима отказа LLM, каждый — sentinel в домене:

| HTTP / ситуация | sentinel | exit code |
|---|---|---|
| 429 | `ErrLLMRateLimited` | 2 |
| 5xx, сеть, decode error | `ErrLLMUnavailable` | 2 |
| `usage.total_tokens > limit` | `ErrLLMBudgetExceeded` | 2 |

Env-переменная не задана → `ErrLLMUnavailable` (fail-fast до HTTP).
Парсинг verdict провалился → `ErrLLMUnavailable` с деталями.

Все три проверяются в компонентных тестах через отдельные режимы стаба.

---

## Стаб для компонентных тестов

Стаб — отдельный HTTP-сервис в Docker Compose, реализующий
тот же `POST /v1/chat/completions` эндпоинт. Не in-code мок.

**Переключение режима** через `POST /control {"mode":"..."}`.
Режимы соответствуют различимым режимам отказа из контракта:

```
healthy        → 200, все роли PASS, разные score
mixed          → 200, смешанные вердикты (PARTIAL + FAIL)
rate_limited   → 429 + Retry-After
unavailable    → 503
budget_exceeded → 200 + usage.total_tokens = 100_000_000
markdown_fenced → 200, вердикт обёрнут в ```json...``` (fallback-парсинг)
```

**Различение ролей в стабе** — маркер `role:<key>` в теле промпта.
Дефолтные промпты обязаны нести этот маркер. Стаб ищет его в теле
запроса и возвращает детерминированный вердикт по роли:

```go
func detectRole(r *http.Request) string {
    b, _ := io.ReadAll(r.Body)
    body := strings.ToLower(string(b))
    for _, role := range []string{"maintainer", "consumer", "manager", "agent"} {
        if strings.Contains(body, "role:"+role) {
            return role
        }
    }
    return "maintainer"
}
```

Реальный провайдер маркер игнорирует; стаб реагирует детерминированно.
Это позволяет специфицировать независимость четырёх JTBD-score
и их не-усреднение.

---

## Тестовые данные: фиксировать реальные ответы модели

При первом успешном прогоне на реальной модели — сохранить ответ в
`component-tests/testdata/real-responses/<repo>-<slice>.json`.

Формат файла:

```json
{
  "_meta": {
    "source": "real Sonnet (claude-sonnet-4-6)",
    "repo": "ubik-life/passkey-demo-api",
    "docs_checked": ["README.md", "CLAUDE.md", "CONTRIBUTING.md", "AGENTS.md"],
    "date": "YYYY-MM-DD"
  },
  "jtbd": { ... }
}
```

На основе этих данных создать отдельный режим стаба (например, `passkey`,
`bad_repo`) — стаб возвращает зафиксированные вердикты детерминированно.
Это даёт сценарий, который воспроизводит поведение реальной модели
без сетевых вызовов.

Покрыть минимум два варианта реальных данных:
- **Хорошая репа** (часть ролей FAIL/PARTIAL — это норма, не провал)
- **Плохая репа** (все роли FAIL, score 2-3, gaps про отсутствие реального содержимого)

---

## Чеклист реализации

- [ ] `baseURL` включает `/v1` (не просто домен)
- [ ] Auth: `Authorization: Bearer`, не `x-api-key`
- [ ] `response_format.type = "json_schema"` с `strict: true` и `additionalProperties: false`
- [ ] Fallback `extractJSON` после `response_format` (второй эшелон)
- [ ] Ключ читается из env по имени из конфига, не хардкод
- [ ] Fail-fast на отсутствующий ключ до I/O
- [ ] Три sentinel-ошибки: `ErrLLMRateLimited`, `ErrLLMUnavailable`, `ErrLLMBudgetExceeded`
- [ ] Пауза между последовательными вызовами (`time.Sleep` между промптами)
- [ ] Список docs в конфиге, не «все .md подряд»
- [ ] Стаб различает роли по маркеру `role:<key>` в промпте
- [ ] Реальные ответы сохранены в `testdata/real-responses/`
- [ ] Сценарии для хорошей и плохой репы зафиксированы в `.feature`
