Feature: fitness — L5 JTBD-пригодность (LLM через стаб)

  # Контракт: report.schema.json (jtbd — ЧЕТЫРЕ независимых результата, каждый
  # {status,score,gaps}; errors[].integration) + cli.md (--config YAML, ключ из
  # env через llm.api_key_env, режимы отказа LLM, JTBD FAIL → код 1).
  # Стаб различает вердикт по роли (маркер "role:<key>" в промпте) — это даёт
  # специфицировать независимость и не-усреднение четырёх score.

  Scenario: стаб отвечает — четыре потребителя оценены, все PASS
    Given LLM-стаб в режиме "healthy"
    When запускаю "fitness" на репозитории "repo-good"
    Then код возврата 0
    And отчёт содержит JSON-поле "command" со значением "fitness"
    And отчёт содержит JSON-поле "jtbd.maintainer.status" со значением "PASS"
    And отчёт содержит непустое JSON-поле "jtbd.maintainer.score"
    And отчёт содержит JSON-поле "jtbd.maintainer.gaps"
    And отчёт содержит JSON-поле "jtbd.consumer.status" со значением "PASS"
    And отчёт содержит JSON-поле "jtbd.manager.status" со значением "PASS"
    And отчёт содержит JSON-поле "jtbd.agent.status" со значением "PASS"

  Scenario: смешанные вердикты — score независимы и не усредняются
    Given LLM-стаб в режиме "mixed"
    When запускаю "fitness" на репозитории "repo-good"
    Then код возврата 1
    And отчёт содержит JSON-поле "jtbd.agent.status" со значением "FAIL"
    And отчёт содержит непустое JSON-поле "jtbd.agent.gaps"
    And отчёт содержит JSON-поле "jtbd.consumer.status" со значением "PARTIAL"
    And отчёт содержит JSON-поле "jtbd.maintainer.status" со значением "PASS"
    And отчёт содержит JSON-поле "jtbd.manager.status" со значением "PASS"

  Scenario: LLM ограничивает частоту
    Given LLM-стаб в режиме "rate_limited"
    When запускаю "fitness" на репозитории "repo-good"
    Then код возврата 2
    And в errors[] есть error.code "llm_rate_limited" с integration "LLMClient"

  Scenario: LLM недоступен (сетевой отказ)
    Given LLM-стаб в режиме "unavailable"
    When запускаю "fitness" на репозитории "repo-good"
    Then код возврата 2
    And в errors[] есть error.code "llm_unavailable" с integration "LLMClient"

  Scenario: бюджет превышен
    Given LLM-стаб в режиме "budget_exceeded"
    When запускаю "fitness" на репозитории "repo-good"
    Then код возврата 2
    And в errors[] есть error.code "llm_budget_exceeded" с integration "LLMClient"

  Scenario: ключ LLM не задан в окружении — LLM не вызывается
    Given LLM-стаб в режиме "healthy"
    And ключ LLM не задан в окружении
    When запускаю "fitness" на репозитории "repo-good"
    Then код возврата 2
    And в errors[] есть error.code "llm_unavailable" с integration "LLMClient"

  Scenario: плохая документация — все четыре роли FAIL
    # Вердикты зафиксированы реальным Sonnet (2026-05-30): repo-bad содержит только
    # README.md с канцелярским placeholder-текстом, остальные docs отсутствуют.
    # Источник: testdata/real-responses/repo-bad-fitness.json.
    Given LLM-стаб в режиме "bad_repo"
    When запускаю "fitness" на репозитории "repo-bad"
    Then код возврата 1
    And отчёт содержит JSON-поле "jtbd.maintainer.status" со значением "FAIL"
    And отчёт содержит непустое JSON-поле "jtbd.maintainer.gaps"
    And отчёт содержит JSON-поле "jtbd.consumer.status" со значением "FAIL"
    And отчёт содержит JSON-поле "jtbd.manager.status" со значением "FAIL"
    And отчёт содержит JSON-поле "jtbd.agent.status" со значением "FAIL"

  Scenario: реальные docs passkey-demo-api — consumer FAIL, остальные PASS
    # Вердикты зафиксированы реальным Sonnet (2026-05-30).
    # Данные: testdata/repo-passkey/ + testdata/real-responses/passkey-demo-api-fitness.json.
    Given LLM-стаб в режиме "passkey"
    When запускаю "fitness" на репозитории "repo-passkey"
    Then код возврата 1
    And отчёт содержит JSON-поле "jtbd.consumer.status" со значением "FAIL"
    And отчёт содержит непустое JSON-поле "jtbd.consumer.gaps"
    And отчёт содержит JSON-поле "jtbd.maintainer.status" со значением "PASS"
    And отчёт содержит JSON-поле "jtbd.manager.status" со значением "PASS"
    And отчёт содержит JSON-поле "jtbd.agent.status" со значением "PASS"

  Scenario: LLM оборачивает JSON в markdown-блок — клиент обязан распарсить
    Given LLM-стаб в режиме "markdown_fenced"
    When запускаю "fitness" на репозитории "repo-good"
    Then код возврата 0
    And отчёт содержит JSON-поле "jtbd.maintainer.status" со значением "PASS"

  Scenario: битый файл --config
    Given битый файл конфигурации
    When запускаю "fitness" на репозитории "repo-good"
    Then код возврата 2
    And в errors[] есть error.code "config_invalid"
