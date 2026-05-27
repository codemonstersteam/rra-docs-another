# Образ заглушки LLM (OpenAI-совместимый эндпоинт) для компонентных тестов.
# Контекст сборки — корень репозитория (см. docker-compose.test.yml).

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY component-tests/go.mod component-tests/go.sum ./
RUN go mod download
COPY component-tests/llm-stub ./llm-stub
RUN CGO_ENABLED=0 go build -o /llm-stub ./llm-stub

FROM alpine:3.20
COPY --from=build /llm-stub /llm-stub
EXPOSE 8080
ENTRYPOINT ["/llm-stub"]
