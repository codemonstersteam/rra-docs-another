Feature: drift — L6 дрейф документации (ядро L6a, без ИИ)

  Scenario: опрятный репозиторий — дока согласована
    When запускаю "drift" на репозитории "repo-good"
    Then код возврата 0
    And отчёт содержит JSON-поле "layers.L6.status" со значением "pass"

  Scenario: битая ссылка — блокирующий дрейф
    When запускаю "drift" на репозитории "repo-bad"
    Then код возврата 1
    And отчёт содержит JSON-поле "layers.L6.status" со значением "fail"

  Scenario: путь не существует
    When запускаю "drift" на репозитории "no-such-repo"
    Then код возврата 2
    And в errors[] есть error.code "path_not_found"
