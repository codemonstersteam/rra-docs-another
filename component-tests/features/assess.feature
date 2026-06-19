Feature: assess — весь пайплайн L1–L6, дешёвое-первым; L5 при наличии документации

  Scenario: опрятный репозиторий — четыре JTBD PASS
    Given LLM-стаб в режиме "healthy"
    When запускаю "assess" на репозитории "repo-good"
    Then код возврата 0
    And отчёт содержит JSON-поле "jtbd.maintainer.status" со значением "PASS"
    And отчёт содержит JSON-поле "jtbd.consumer.status" со значением "PASS"
    And отчёт содержит JSON-поле "jtbd.manager.status" со значением "PASS"
    And отчёт содержит JSON-поле "jtbd.agent.status" со значением "PASS"

  Scenario: документация есть, статика частично провалена, ИИ годен — итог PARTIAL
    # ADR 0004 (смягчение гейта L5): FAIL на L4 не пропускает L5, а ограничивает
    # итог сверху до PARTIAL — capL5ByL4. Роль с провалом статики, но годная по ИИ,
    # даёт PARTIAL (не FAIL и не skipped); код возврата 0.
    Given LLM-стаб в режиме "healthy"
    When запускаю "assess" на репозитории "repo-soft"
    Then код возврата 0
    And отчёт содержит JSON-поле "jtbd.agent.status" со значением "PARTIAL"

  Scenario: битый репозиторий — ИИ подтверждает провал
    # Документация есть, но и статика, и ИИ её проваливают → JTBD FAIL → код 1.
    # L5 теперь реально запускается (раньше пропускался по short-circuit).
    Given LLM-стаб в режиме "bad_repo"
    When запускаю "assess" на репозитории "repo-bad"
    Then код возврата 1

  Scenario: путь не существует
    When запускаю "assess" на репозитории "no-such-repo"
    Then код возврата 2
    And в errors[] есть error.code "path_not_found"
