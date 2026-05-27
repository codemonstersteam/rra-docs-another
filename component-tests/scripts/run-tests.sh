#!/usr/bin/env bash
# Запуск компонентных тестов rra-docs-another.
#
#   ./scripts/run-tests.sh              # smoke + реализованные сценарии (@wip пропущены)
#   GODOG_TAGS='~@never' ./scripts/run-tests.sh   # прогнать всё, включая @wip (красное до реализации)
#
# Раннер запускается ВНУТРИ Docker (требование изоляции). Никаких `go test`
# с хоста — поведение разойдётся между CI и машиной разработчика.

set -euo pipefail
cd "$(dirname "$0")/.."

COMPOSE=(-f docker-compose.test.yml)

cleanup() {
  docker compose "${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "==> building images..."
docker compose "${COMPOSE[@]}" build

echo "==> running component tests..."
# --exit-code-from tester возвращает именно код раннера; --abort-on-container-exit
# гасит llm-stub, когда раннер закончил.
docker compose "${COMPOSE[@]}" up \
  --abort-on-container-exit \
  --exit-code-from tester
