@wip
Feature: assess — весь пайплайн L1–L6, дешёвое-первым, short-circuit

  Scenario: опрятный репозиторий — четыре JTBD PASS
    Given LLM-стаб в режиме "healthy"
    When запускаю "assess" на репозитории "repo-good"
    Then код возврата 0
    And отчёт содержит JSON-поле "jtbd.maintainer.status" со значением "PASS"
    And отчёт содержит JSON-поле "jtbd.consumer.status" со значением "PASS"
    And отчёт содержит JSON-поле "jtbd.manager.status" со значением "PASS"
    And отчёт содержит JSON-поле "jtbd.agent.status" со значением "PASS"

  Scenario: битый репозиторий — есть проваленный JTBD
    Given LLM-стаб в режиме "healthy"
    When запускаю "assess" на репозитории "repo-bad"
    Then код возврата 1

  Scenario: путь не существует
    When запускаю "assess" на репозитории "no-such-repo"
    Then код возврата 2
    And в errors[] есть error.code "path_not_found"
