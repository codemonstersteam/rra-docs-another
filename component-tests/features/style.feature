@wip
Feature: style — L2 стиль (Vale, markdownlint)

  # Given-степы «линтеры недоступны» / «линтер завершается с ошибкой» реализуются
  # в слайсе S4 (style) вместе с установкой Vale/markdownlint в образ раннера.

  Scenario: опрятный репозиторий
    When запускаю "style" на репозитории "repo-good"
    Then код возврата 0
    And отчёт содержит JSON-поле "layers.L2.status" со значением "pass"

  Scenario: линтер не установлен
    Given линтеры недоступны
    When запускаю "style" на репозитории "repo-good"
    Then код возврата 2
    And в errors[] есть error.code "tool_missing"

  Scenario: линтер завершается с ошибкой
    Given линтер завершается с ошибкой
    When запускаю "style" на репозитории "repo-good"
    Then код возврата 2
    And в errors[] есть error.code "tool_failed"
