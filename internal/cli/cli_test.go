package cli

import (
	"bytes"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, err bytes.Buffer
	code = Run(args, &out, &err)
	return code, out.String(), err.String()
}

func TestVersion(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v"} {
		code, stdout, _ := run(t, arg)
		if code != 0 {
			t.Errorf("%s: код %d, ожидали 0", arg, code)
		}
		if strings.TrimSpace(stdout) != Version {
			t.Errorf("%s: stdout %q, ожидали %q", arg, strings.TrimSpace(stdout), Version)
		}
	}
}

func TestHelp(t *testing.T) {
	code, stdout, _ := run(t, "help")
	if code != 0 {
		t.Errorf("код %d, ожидали 0", code)
	}
	if !strings.Contains(stdout, "Использование:") {
		t.Errorf("usage не выведен на stdout: %q", stdout)
	}
}

func TestNoArgs(t *testing.T) {
	code, _, stderr := run(t)
	if code != 2 {
		t.Errorf("код %d, ожидали 2", code)
	}
	if !strings.Contains(stderr, "Использование:") {
		t.Errorf("usage не выведен на stderr: %q", stderr)
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, stderr := run(t, "bogus")
	if code != 2 {
		t.Errorf("код %d, ожидали 2", code)
	}
	if !strings.Contains(stderr, "неизвестная команда") {
		t.Errorf("нет сообщения об ошибке: %q", stderr)
	}
}

func TestKnownSubcommandNotImplemented(t *testing.T) {
	for _, c := range subcommands {
		code, _, stderr := run(t, c)
		if code != 2 {
			t.Errorf("%s: код %d, ожидали 2", c, code)
		}
		if !strings.Contains(stderr, "ещё не реализована") {
			t.Errorf("%s: ожидали «ещё не реализована», получили %q", c, stderr)
		}
	}
}
