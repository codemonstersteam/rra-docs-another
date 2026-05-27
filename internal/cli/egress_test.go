package cli_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/cli"
	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// ── buildErrorReport ─────────────────────────────────────────────────────────

func TestBuildErrorReport_pathNotFound(t *testing.T) {
	req := domain.Request{Command: "structure", Path: "/no-such"}
	err := fmt.Errorf("%w: /no-such", domain.ErrPathNotFound)
	report := cli.ExportBuildErrorReport(req, err)
	if len(report.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(report.Errors))
	}
	if report.Errors[0].Code != "path_not_found" {
		t.Errorf("expected path_not_found, got %s", report.Errors[0].Code)
	}
	if report.Command != "structure" {
		t.Errorf("expected command=structure, got %s", report.Command)
	}
}

func TestBuildErrorReport_configInvalid(t *testing.T) {
	req := domain.Request{Command: "structure"}
	err := fmt.Errorf("%w: bad file", domain.ErrConfigInvalid)
	report := cli.ExportBuildErrorReport(req, err)
	if len(report.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(report.Errors))
	}
	if report.Errors[0].Code != "config_invalid" {
		t.Errorf("expected config_invalid, got %s", report.Errors[0].Code)
	}
	if report.Errors[0].Integration != nil {
		t.Errorf("expected nil integration, got %v", *report.Errors[0].Integration)
	}
}

// ── exitCode ─────────────────────────────────────────────────────────────────

func TestExitCode_noErrors(t *testing.T) {
	report := domain.Report{}
	if code := cli.ExportExitCode(report); code != 0 {
		t.Errorf("expected 0, got %d", code)
	}
}

func TestExitCode_withErrors(t *testing.T) {
	report := domain.Report{Errors: []domain.Error{{Code: "path_not_found", Message: "x"}}}
	if code := cli.ExportExitCode(report); code != 2 {
		t.Errorf("expected 2, got %d", code)
	}
}

func TestExitCode_withBlocker(t *testing.T) {
	report := domain.Report{
		Violations: []domain.Violation{{Layer: "L3", Severity: "blocker", Code: "missing_readme", File: "README.md", Message: "x"}},
	}
	if code := cli.ExportExitCode(report); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestExitCode_l1BlockerNeverOne(t *testing.T) {
	// L1 нарушения не дают код 1.
	report := domain.Report{
		Violations: []domain.Violation{{Layer: "L1", Severity: "blocker", Code: "readability", File: "README.md", Message: "x"}},
	}
	if code := cli.ExportExitCode(report); code != 0 {
		t.Errorf("expected 0 for L1 blocker, got %d", code)
	}
}

func TestExitCode_jtbdFail(t *testing.T) {
	report := domain.Report{
		JTBD: map[string]domain.JTBDResult{
			"maintainer": {Status: "FAIL", Score: 10, Gaps: []string{"no arch"}},
		},
	}
	if code := cli.ExportExitCode(report); code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

// Проверяем что errors.Is работает через wrapping.
func TestMapError_wrappedSentinel(t *testing.T) {
	err := fmt.Errorf("context: %w", domain.ErrReadError)
	report := cli.ExportBuildErrorReport(domain.Request{Command: "structure"}, err)
	if report.Errors[0].Code != "read_error" {
		t.Errorf("expected read_error, got %s", report.Errors[0].Code)
	}
}

func TestMapError_unknownError(t *testing.T) {
	err := errors.New("something weird")
	report := cli.ExportBuildErrorReport(domain.Request{Command: "structure"}, err)
	// Fallback — read_error.
	if report.Errors[0].Code != "read_error" {
		t.Errorf("expected read_error fallback, got %s", report.Errors[0].Code)
	}
}
