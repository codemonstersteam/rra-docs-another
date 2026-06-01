# Сервис

Сервис авторизации по passkey. Решает задачу беспарольного входа.

## Запуск

`go run ./cmd/api` или `docker compose up`.

## API

- `POST /v1/registrations` — начать регистрацию.
- `POST /v1/sessions` — начать сессию.

## Что умеет

Регистрация и аутентификация по WebAuthn. Выдаёт JWT.
