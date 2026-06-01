Feature: readability — L1 читаемость

  Scenario: опрятный репозиторий
    When запускаю "readability" на репозитории "repo-good-readability"
    Then код возврата 0
    And отчёт содержит JSON-поле "layers.L1.status" со значением "pass"

  Scenario: низкая читаемость не блокирует (L1 — порог-warning, не блок)
    When запускаю "readability" на репозитории "repo-bad-readability"
    Then код возврата 0
    And отчёт содержит JSON-поле "layers.L1.status" со значением "warn"

  Scenario: путь не существует
    When запускаю "readability" на репозитории "no-such-repo"
    Then код возврата 2
    And в errors[] есть error.code "path_not_found"
