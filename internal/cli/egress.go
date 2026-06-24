package cli

import (
	"errors"
	"fmt"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// buildErrorReport — чистая логика: разворачивает доменную ошибку в Report{Errors:[…]}.
// Маппинг sentinel → error.code по таблице messages.md.
func buildErrorReport(req domain.Request, err error) domain.Report {
	e := mapError(err)
	return domain.Report{
		SchemaVersion: "1.0",
		Tool:          "rra-docs-another",
		Command:       req.Command,
		Target: domain.ReportTarget{
			Path: req.Path,
		},
		Errors: []domain.Error{e},
	}
}

// mapError переводит доменную sentinel-ошибку в Error{Code,Integration,Message}.
func mapError(err error) domain.Error {
	type entry struct {
		sentinel    error
		code        string
		integration *string
	}

	repoStore := "RepoStore"
	linterRunner := "LinterRunner"
	llmClient := "LLMClient"
	reportSink := "ReportSink"

	table := []entry{
		{domain.ErrPathNotFound, "path_not_found", &repoStore},
		{domain.ErrReadError, "read_error", &repoStore},
		{domain.ErrConfigInvalid, "config_invalid", nil},
		{domain.ErrToolMissing, "tool_missing", &linterRunner},
		{domain.ErrToolFailed, "tool_failed", &linterRunner},
		{domain.ErrLLMRateLimited, "llm_rate_limited", &llmClient},
		{domain.ErrLLMUnavailable, "llm_unavailable", &llmClient},
		{domain.ErrLLMBudgetExceeded, "llm_budget_exceeded", &llmClient},
		{domain.ErrUnknownFormat, "format_invalid", nil},
		{domain.ErrReportWrite, "report_write_failed", &reportSink},
	}

	for _, e := range table {
		if errors.Is(err, e.sentinel) {
			return domain.Error{
				Code:        e.code,
				Integration: e.integration,
				Message:     err.Error(),
			}
		}
	}

	// Неизвестная ошибка — read_error как fallback (ошибка I/O).
	return domain.Error{
		Code:    "read_error",
		Message: fmt.Sprintf("неизвестная ошибка: %v", err),
	}
}

// exitCode вычисляет код возврата по содержимому отчёта.
// 2 — есть Errors; 1 — есть blocker-нарушение или JTBD FAIL; 0 — иначе.
// Нарушения layer:"L1" никогда не дают код 1 (по контракту L1).
func exitCode(report domain.Report) int {
	if len(report.Errors) > 0 {
		return 2
	}
	for _, v := range report.Violations {
		if v.Layer == "L1" {
			continue
		}
		if v.Severity == "blocker" {
			return 1
		}
	}
	for _, j := range report.JTBD {
		if j.Status == "FAIL" {
			return 1
		}
	}
	return 0
}
