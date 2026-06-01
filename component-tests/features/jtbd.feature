Feature: jtbd — L4 JTBD-присутствие

  Scenario: опрятный репозиторий — все четыре JTBD PASS
    When запускаю "jtbd" на репозитории "repo-good-jtbd"
    Then код возврата 0
    And отчёт содержит JSON-поле "jtbd.maintainer.status" со значением "PASS"
    And отчёт содержит JSON-поле "jtbd.consumer.status" со значением "PASS"
    And отчёт содержит JSON-поле "jtbd.manager.status" со значением "PASS"
    And отчёт содержит JSON-поле "jtbd.agent.status" со значением "PASS"

  Scenario: битый репозиторий — есть проваленный JTBD
    When запускаю "jtbd" на репозитории "repo-bad-jtbd"
    Then код возврата 1
    And отчёт содержит JSON-поле "jtbd.agent.status" со значением "FAIL"

  Scenario: путь не существует
    When запускаю "jtbd" на репозитории "no-such-repo"
    Then код возврата 2
    And в errors[] есть error.code "path_not_found"
