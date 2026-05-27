@wip
Feature: fitness — L5 JTBD-пригодность (LLM через стаб)

  Scenario: стаб отвечает — пригодность оценена
    Given LLM-стаб в режиме "healthy"
    When запускаю "fitness" на репозитории "repo-good"
    Then код возврата 0
    And отчёт содержит JSON-поле "jtbd.maintainer.status" со значением "PASS"

  Scenario: LLM ограничивает частоту
    Given LLM-стаб в режиме "rate_limited"
    When запускаю "fitness" на репозитории "repo-good"
    Then код возврата 2
    And в errors[] есть error.code "llm_rate_limited"

  Scenario: LLM недоступен
    Given LLM-стаб в режиме "unavailable"
    When запускаю "fitness" на репозитории "repo-good"
    Then код возврата 2
    And в errors[] есть error.code "llm_unavailable"

  Scenario: бюджет превышен
    Given LLM-стаб в режиме "budget_exceeded"
    When запускаю "fitness" на репозитории "repo-good"
    Then код возврата 2
    And в errors[] есть error.code "llm_budget_exceeded"
