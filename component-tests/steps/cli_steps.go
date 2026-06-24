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
	ctx.Step(`^запускаю "([^"]+)" на репозитории "([^"]+)" с выводом в "([^"]*)"$`, w.runOnRepoWithOut)
	ctx.Step(`^LLM-стаб в режиме "([^"]+)"$`, w.setStubMode)
	ctx.Step(`^ключ LLM не задан в окружении$`, w.setNoLLMKey)
	ctx.Step(`^битый файл конфигурации$`, w.setBrokenConfig)

	// Проверки
	ctx.Step(`^код возврата (\d+)$`, w.assertExit)
	ctx.Step(`^отчёт содержит JSON-поле "([^"]+)" со значением "([^"]*)"$`, w.assertField)
	ctx.Step(`^отчёт содержит непустое JSON-поле "([^"]+)"$`, w.assertFieldNonEmpty)
	ctx.Step(`^отчёт содержит JSON-поле "([^"]+)"$`, w.assertFieldPresent)
	ctx.Step(`^в errors\[\] есть error\.code "([^"]+)"$`, w.assertErrorCode)
	ctx.Step(`^в errors\[\] есть error\.code "([^"]+)" с integration "([^"]+)"$`, w.assertErrorCodeIntegration)
	ctx.Step(`^stderr содержит "([^"]+)"$`, w.assertStderr)
}

func (w *World) runRaw(ctx context.Context, args string) error {
	return w.run(ctx, strings.Fields(args))
}

// runOnRepoWithOut запускает подкоманду с флагом --out <path> — для проверки
// записи отчёта (egress). Путь берётся как есть (может быть заведомо
// недоступным — сценарий отказа записи).
func (w *World) runOnRepoWithOut(ctx context.Context, cmd, fixture, out string) error {
	args := []string{cmd, filepath.Join(w.testdataDir, fixture), "--format", "json", "--out", out}
	return w.run(ctx, args)
}

func (w *World) runOnRepo(ctx context.Context, cmd, fixture string) error {
	args := []string{cmd, filepath.Join(w.testdataDir, fixture), "--format", "json"}
	env := []string{}
	if cmd == "fitness" || cmd == "assess" || cmd == "drift" {
		args = append(args, "--llm-provider", "openai", "--llm-base-url", w.llmBaseURL)
		if !w.dropLLMKey {
			env = append(env, "OPENAI_API_KEY=test")
		}
	}
	if w.configPath != "" {
		args = append(args, "--config", w.configPath)
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

// setNoLLMKey помечает запуск как «без ключа в env»: runOnRepo не прокинет
// OPENAI_API_KEY → NewLLMConfig падает в ErrLLMUnavailable до вызова LLM.
func (w *World) setNoLLMKey() error {
	w.dropLLMKey = true
	return nil
}

// setBrokenConfig направляет --config на заведомо битый YAML-файл фикстуры,
// чтобы загрузчик конфига вернул config_invalid.
func (w *World) setBrokenConfig() error {
	w.configPath = filepath.Join(w.testdataDir, "broken-config.yml")
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

// assertErrorCodeIntegration проверяет, что в errors[] есть запись с заданными
// error.code И errors[].integration (контракт схемы: где произошёл отказ).
func (w *World) assertErrorCodeIntegration(code, integration string) error {
	got, ok := w.field("errors")
	if !ok {
		return fmt.Errorf("в отчёте нет errors[]")
	}
	list, ok := got.([]any)
	if !ok {
		return fmt.Errorf("errors не массив")
	}
	for _, e := range list {
		m, ok := e.(map[string]any)
		if !ok || scalar(m["code"]) != code {
			continue
		}
		if g := scalar(m["integration"]); g != integration {
			return fmt.Errorf("error.code=%q: integration=%q, ожидали %q", code, g, integration)
		}
		return nil
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
