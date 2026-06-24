Feature: egress — общий вывод отчёта (флаг --out)

  # Сквозная поверхность общего egress. Развилки stdout|файл × json|md —
  # юниты (resolveDestination/renderReport, см. docs/design/assess/egress.md);
  # в компонент входит только режим ОТКАЗА записи (правило различимости).

  Scenario: запись отчёта в недоступный путь — отказ I/O
    # --out на несуществующий каталог → os.WriteFile падает → ErrReportWrite.
    # Отчёт об ошибке (фолбэк) печатается в stdout как JSON; код возврата 2.
    When запускаю "structure" на репозитории "repo-good-structure" с выводом в "/nonexistent-dir/report.json"
    Then код возврата 2
    And в errors[] есть error.code "report_write_failed" с integration "ReportSink"
