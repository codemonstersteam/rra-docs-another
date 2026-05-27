# AGENTS.md — контекст для ИИ-агента

Сервис расчёта виджетов. Точка входа — `cmd/api`. Бизнес-логика — `internal/plan`,
хранилище — `internal/store` (PostgreSQL).

Чтобы изменить алгоритм раскроя: правь `internal/plan/cut.go`, тест — `cut_test.go`.
Запуск: `docker compose up`; проверка: `go test ./...`.
