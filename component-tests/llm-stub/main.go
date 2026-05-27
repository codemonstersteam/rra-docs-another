// Command llm-stub — управляемая заглушка OpenAI-совместимого LLM-эндпоинта для
// компонентных тестов. Не in-code мок: тул реально ходит по сети к этому
// процессу за тем же проводным контрактом, что и к настоящему провайдеру.
//
// Эндпоинты:
//
//	GET  /healthz                — healthcheck для docker-compose.
//	POST /control  {"mode":"…"}  — переключить режим ответа (in-memory).
//	POST /v1/chat/completions    — ответ согласно текущему режиму.
//
// Режимы (= различимые режимы отказа LLMClient из контракта):
//
//	healthy           — 200 + канонный verdict;
//	rate_limited      — 429 + Retry-After      → llm_rate_limited;
//	unavailable       — 503                     → llm_unavailable;
//	budget_exceeded   — 200 + огромный usage    → llm_budget_exceeded.
//
// Режим переключается степом до запуска бинаря. Сценарии godog идут
// последовательно, поэтому единственного инстанса с mutable-режимом достаточно.
package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sync/atomic"
)

func main() {
	addr := os.Getenv("STUB_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	var mode atomic.Value // string
	mode.Store("healthy")

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/control", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Mode string `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Mode == "" {
			http.Error(w, `{"error":"mode required"}`, http.StatusBadRequest)
			return
		}
		mode.Store(body.Mode)
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, _ *http.Request) {
		switch mode.Load().(string) {
		case "rate_limited":
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":{"type":"rate_limit_error"}}`, http.StatusTooManyRequests)
		case "unavailable":
			http.Error(w, `{"error":{"type":"server_error"}}`, http.StatusServiceUnavailable)
		case "budget_exceeded":
			writeCompletion(w, `{"status":"PASS","score":90,"gaps":[]}`, 100_000_000)
		default: // healthy
			writeCompletion(w, `{"status":"PASS","score":90,"gaps":[]}`, 42)
		}
	})

	if err := http.ListenAndServe(addr, mux); err != nil {
		panic(err)
	}
}

// writeCompletion печатает минимальный OpenAI chat.completion с заданным
// content и числом потраченных токенов.
func writeCompletion(w http.ResponseWriter, content string, totalTokens int) {
	resp := map[string]any{
		"id":      "stub",
		"object":  "chat.completion",
		"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": content}, "finish_reason": "stop"}},
		"usage":   map[string]any{"total_tokens": totalTokens},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
