// Package steps содержит godog-степы компонентных тестов rra-docs-another.
//
// World — состояние сценария: где бинарь-SUT, как достучаться до LLM-стаба, и
// результат последнего запуска (код возврата, stdout/stderr, разобранный отчёт).
// SUT — это CLI: каждый When exec-ает бинарь внутри контейнера-раннера, до
// llm-stub ходит по сети по имени сервиса. Состояние не утекает между
// сценариями (World пересоздаётся в InitializeScenario).
package steps

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cucumber/godog"
)

type World struct {
	binPath       string // путь к собранному бинарю (RRA_BIN)
	llmBaseURL    string // OpenAI-совместимый base-url стаба (LLM_STUB_BASE_URL)
	llmControlURL string // control-эндпоинт стаба (LLM_STUB_CONTROL_URL)
	testdataDir   string // корень фикстур относительно ./steps

	lastExit   int
	lastStdout []byte
	lastStderr []byte
	report     map[string]any // разобранный JSON-отчёт последнего запуска
}

func newWorld() *World {
	return &World{
		binPath:       getenv("RRA_BIN", "rra-docs-another"),
		llmBaseURL:    getenv("LLM_STUB_BASE_URL", "http://llm-stub:8080/v1"),
		llmControlURL: getenv("LLM_STUB_CONTROL_URL", "http://llm-stub:8080/control"),
		testdataDir:   getenv("RRA_TESTDATA", "../testdata"),
	}
}

func (w *World) beforeScenario(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
	w.lastExit = 0
	w.lastStdout = nil
	w.lastStderr = nil
	w.report = nil
	return ctx, nil
}

// run исполняет бинарь с args, фиксирует код возврата и stdout/stderr.
func (w *World) run(ctx context.Context, args []string, extraEnv ...string) error {
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(runCtx, w.binPath, args...)
	cmd.Env = append(os.Environ(), extraEnv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	w.lastStdout = stdout.Bytes()
	w.lastStderr = stderr.Bytes()

	var exitErr *exec.ExitError
	switch {
	case err == nil:
		w.lastExit = 0
	case errors.As(err, &exitErr):
		w.lastExit = exitErr.ExitCode()
	default:
		return fmt.Errorf("запуск %q: %w (stderr: %s)", w.binPath, err, stderr.String())
	}

	// Лучшая попытка разобрать stdout как JSON-отчёт (для --format json).
	w.report = nil
	if bytes.HasPrefix(bytes.TrimSpace(w.lastStdout), []byte("{")) {
		_ = json.Unmarshal(w.lastStdout, &w.report)
	}
	return nil
}

// field достаёт значение по dotted-пути из разобранного отчёта.
func (w *World) field(path string) (any, bool) {
	if w.report == nil {
		return nil, false
	}
	var cur any = w.report
	for _, key := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[key]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
