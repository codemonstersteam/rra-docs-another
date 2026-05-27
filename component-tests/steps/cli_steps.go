package steps

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cucumber/godog"
)

// registerCLISteps регистрирует все степы. Словарь — аналог HTTP-степов
// эталона (passkey), но для CLI: вместо кода ответа — код возврата, вместо
// тела ответа — отчёт по report.schema.json.
func (w *World) registerCLISteps(ctx *godog.ScenarioContext) {
	// Запуск
	ctx.Step(`^запускаю бинарь с аргументами "([^"]*)"$`, w.runRaw)
	ctx.Step(`^запускаю "([^"]+)" на репозитории "([^"]+)"$`, w.runOnRepo)
	ctx.Step(`^LLM-стаб в режиме "([^"]+)"$`, w.setStubMode)

	// Проверки
	ctx.Step(`^код возврата (\d+)$`, w.assertExit)
	ctx.Step(`^отчёт содержит JSON-поле "([^"]+)" со значением "([^"]*)"$`, w.assertField)
	ctx.Step(`^отчёт содержит непустое JSON-поле "([^"]+)"$`, w.assertFieldNonEmpty)
	ctx.Step(`^отчёт содержит JSON-поле "([^"]+)"$`, w.assertFieldPresent)
	ctx.Step(`^в errors\[\] есть error\.code "([^"]+)"$`, w.assertErrorCode)
	ctx.Step(`^stderr содержит "([^"]+)"$`, w.assertStderr)
}

func (w *World) runRaw(ctx context.Context, args string) error {
	return w.run(ctx, strings.Fields(args))
}

func (w *World) runOnRepo(ctx context.Context, cmd, fixture string) error {
	args := []string{cmd, filepath.Join(w.testdataDir, fixture), "--format", "json"}
	env := []string{}
	if cmd == "fitness" || cmd == "assess" || cmd == "drift" {
		args = append(args, "--llm-provider", "openai", "--llm-base-url", w.llmBaseURL)
		env = append(env, "OPENAI_API_KEY=test")
	}
	return w.run(ctx, args, env...)
}

func (w *World) setStubMode(mode string) error {
	body := bytes.NewBufferString(fmt.Sprintf(`{"mode":%q}`, mode))
	resp, err := http.Post(w.llmControlURL, "application/json", body)
	if err != nil {
		return fmt.Errorf("LLM-стаб /control: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("LLM-стаб /control: код %d", resp.StatusCode)
	}
	return nil
}

func (w *World) assertExit(code int) error {
	if w.lastExit != code {
		return fmt.Errorf("код возврата %d, ожидали %d (stderr: %s)", w.lastExit, code, strings.TrimSpace(string(w.lastStderr)))
	}
	return nil
}

func (w *World) assertField(path, want string) error {
	got, ok := w.field(path)
	if !ok {
		return fmt.Errorf("поля %q нет в отчёте", path)
	}
	if g := scalar(got); g != want {
		return fmt.Errorf("поле %q = %q, ожидали %q", path, g, want)
	}
	return nil
}

func (w *World) assertFieldPresent(path string) error {
	if _, ok := w.field(path); !ok {
		return fmt.Errorf("поля %q нет в отчёте", path)
	}
	return nil
}

func (w *World) assertFieldNonEmpty(path string) error {
	got, ok := w.field(path)
	if !ok {
		return fmt.Errorf("поля %q нет в отчёте", path)
	}
	if scalar(got) == "" {
		return fmt.Errorf("поле %q пустое", path)
	}
	return nil
}

func (w *World) assertErrorCode(code string) error {
	got, ok := w.field("errors")
	if !ok {
		return fmt.Errorf("в отчёте нет errors[]")
	}
	list, ok := got.([]any)
	if !ok {
		return fmt.Errorf("errors не массив")
	}
	for _, e := range list {
		if m, ok := e.(map[string]any); ok && scalar(m["code"]) == code {
			return nil
		}
	}
	return fmt.Errorf("в errors[] нет error.code=%q", code)
}

func (w *World) assertStderr(substr string) error {
	if !strings.Contains(string(w.lastStderr), substr) {
		return fmt.Errorf("stderr не содержит %q (stderr: %s)", substr, strings.TrimSpace(string(w.lastStderr)))
	}
	return nil
}

// scalar приводит JSON-значение к строке для сравнения. Числа без дробной
// части печатаются как целые (JSON-числа в Go — float64).
func scalar(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%g", x)
	case bool:
		return fmt.Sprintf("%t", x)
	default:
		return fmt.Sprintf("%v", x)
	}
}
