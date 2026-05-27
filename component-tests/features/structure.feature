@wip
Feature: structure — L3 структурная полнота

  Scenario: опрятный репозиторий проходит
    When запускаю "structure" на репозитории "repo-good"
    Then код возврата 0
    And отчёт содержит JSON-поле "command" со значением "structure"
    And отчёт содержит JSON-поле "layers.L3.status" со значением "pass"

  Scenario: битый репозиторий — блокирующее нарушение
    When запускаю "structure" на репозитории "repo-bad"
    Then код возврата 1
    And отчёт содержит JSON-поле "layers.L3.status" со значением "fail"

  Scenario: путь не существует
    When запускаю "structure" на репозитории "no-such-repo"
    Then код возврата 2
    And в errors[] есть error.code "path_not_found"
