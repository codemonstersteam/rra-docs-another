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
// Режимы (= различимые режимы поведения LLMClient из контракта):
//
//	healthy           — 200, по-ролевой вердикт, все PASS (разные score) → код 0;
//	mixed             — 200, по-ролевой вердикт со смешанными статусами
//	                    (consumer PARTIAL, agent FAIL) → код 1; специфицирует
//	                    независимость и не-усреднение четырёх JTBD-score;
//	rate_limited      — 429 + Retry-After      → llm_rate_limited;
//	unavailable       — 503                     → llm_unavailable;
//	budget_exceeded   — 200 + огромный usage    → llm_budget_exceeded.
//
// Различение роли: стаб ищет в теле запроса маркер "role:<key>" (key ∈
// maintainer|consumer|manager|agent). Это контракт между стабом и ДЕФОЛТНЫМИ
// промптами S5 — каждый дефолтный промпт обязан нести свой "role:<key>". Реальный
// провайдер такой маркер просто игнорирует; стаб же реагирует на содержимое
// промпта детерминированно, как реагировала бы настоящая модель.
//
// Режим переключается степом до запуска бинаря. Сценарии godog идут
// последовательно, поэтому единственного инстанса с mutable-режимом достаточно.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		switch m := mode.Load().(string); m {
		case "rate_limited":
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":{"type":"rate_limit_error"}}`, http.StatusTooManyRequests)
		case "unavailable":
			http.Error(w, `{"error":{"type":"server_error"}}`, http.StatusServiceUnavailable)
		case "budget_exceeded":
			writeCompletion(w, `{"status":"PASS","score":90,"gaps":[]}`, 100_000_000)
		default: // healthy | mixed — по-ролевой вердикт
			writeCompletion(w, verdictFor(detectRole(r), m), 42)
		}
	})

	if err := http.ListenAndServe(addr, mux); err != nil {
		panic(err)
	}
}

// detectRole определяет JTBD-роль по маркеру "role:<key>" в теле запроса.
// Дефолтные промпты S5 несут этот маркер. Если маркера нет — fallback maintainer.
func detectRole(r *http.Request) string {
	b, _ := io.ReadAll(r.Body)
	body := strings.ToLower(string(b))
	for _, role := range []string{"maintainer", "consumer", "manager", "agent"} {
		if strings.Contains(body, "role:"+role) {
			return role
		}
	}
	return "maintainer"
}

// verdictFor — канонный JSON-вердикт по роли и режиму.
//
//	healthy — все PASS, разные score (доказывает четыре независимых результата);
//	mixed   — consumer PARTIAL, agent FAIL (доказывает не-усреднение: провал одной
//	          роли не тянет остальные; JTBD FAIL → код 1; gaps протекают наружу).
func verdictFor(role, mode string) string {
	type v struct {
		status string
		score  int
		gaps   string
	}
	healthy := map[string]v{
		"maintainer": {"PASS", 92, "[]"},
		"consumer":   {"PASS", 85, "[]"},
		"manager":    {"PASS", 78, "[]"},
		"agent":      {"PASS", 70, "[]"},
	}
	mixed := map[string]v{
		"maintainer": {"PASS", 88, "[]"},
		"consumer":   {"PARTIAL", 55, `["README не описывает лимиты и интеграции сервиса"]`},
		"manager":    {"PASS", 80, "[]"},
		"agent":      {"FAIL", 30, `["нет карты файлов под задачу для ИИ-агента"]`},
	}
	table := healthy
	if mode == "mixed" {
		table = mixed
	}
	x, ok := table[role]
	if !ok {
		x = table["maintainer"]
	}
	return fmt.Sprintf(`{"status":%q,"score":%d,"gaps":%s}`, x.status, x.score, x.gaps)
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
