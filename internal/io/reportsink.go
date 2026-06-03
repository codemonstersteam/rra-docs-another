package io

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// ReportSink записывает отчёт в stdout или файл.
type ReportSink struct{}

// NewReportSink создаёт ReportSink.
func NewReportSink() ReportSink { return ReportSink{} }

// Write сериализует отчёт в формате format и пишет в out.
// out "-" — stdout, иначе путь к файлу.
func (s ReportSink) Write(report domain.Report, format, out string) error {
	var content string
	switch strings.ToLower(format) {
	case "json", "":
		b, err := json.Marshal(report)
		if err != nil {
			return fmt.Errorf("ReportSink.Write JSON marshal: %w", err)
		}
		content = string(b)
	case "md":
		content = renderMarkdown(report)
	default:
		return fmt.Errorf("ReportSink.Write: неизвестный формат %q", format)
	}

	if out == "" || out == "-" {
		_, err := fmt.Fprintln(os.Stdout, content)
		return err
	}
	return os.WriteFile(out, []byte(content+"\n"), 0o644)
}

// WriteTo записывает отчёт в произвольный io.Writer (для тестов и egress).
func (s ReportSink) WriteTo(report domain.Report, format string, w io.Writer) error {
	var content string
	switch strings.ToLower(format) {
	case "json", "":
		b, err := json.Marshal(report)
		if err != nil {
			return fmt.Errorf("ReportSink.WriteTo JSON marshal: %w", err)
		}
		content = string(b)
	case "md":
		content = renderMarkdown(report)
	default:
		return fmt.Errorf("ReportSink.WriteTo: неизвестный формат %q", format)
	}
	_, err := fmt.Fprintln(w, content)
	return err
}

// renderMarkdown генерирует человекочитаемый Markdown-отчёт.
func renderMarkdown(r domain.Report) string {
	var sb strings.Builder
	sb.WriteString("# rra-docs-another: ")
	sb.WriteString(r.Command)
	sb.WriteString("\n\n")
	sb.WriteString("**Target:** ")
	sb.WriteString(r.Target.Path)
	sb.WriteString("\n\n")

	if len(r.Errors) > 0 {
		sb.WriteString("## Errors\n\n")
		for _, e := range r.Errors {
			sb.WriteString("- **")
			sb.WriteString(e.Code)
			sb.WriteString("**: ")
			sb.WriteString(e.Message)
			sb.WriteString("\n")
		}
		return sb.String()
	}

	if len(r.Layers) > 0 {
		sb.WriteString("## Layers\n\n")
		keys := make([]string, 0, len(r.Layers))
		for k := range r.Layers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			l := r.Layers[k]
			sb.WriteString("### ")
			sb.WriteString(k)
			sb.WriteString(": ")
			sb.WriteString(l.Name)
			sb.WriteString("\n\n")
			sb.WriteString("Status: **")
			sb.WriteString(l.Status)
			sb.WriteString("**\n\n")
			if l.Summary != "" {
				sb.WriteString(l.Summary)
				sb.WriteString("\n\n")
			}
		}
	}

	if len(r.JTBD) > 0 {
		sb.WriteString("## JTBD\n\n")
		roles := make([]string, 0, len(r.JTBD))
		for role := range r.JTBD {
			roles = append(roles, role)
		}
		sort.Strings(roles)
		for _, role := range roles {
			j := r.JTBD[role]
			sb.WriteString("### ")
			sb.WriteString(role)
			sb.WriteString("\n\n")
			sb.WriteString("Status: **")
			sb.WriteString(j.Status)
			sb.WriteString("** | Score: ")
			sb.WriteString(fmt.Sprintf("%d", j.Score))
			sb.WriteString("\n\n")
			if len(j.Gaps) > 0 {
				sb.WriteString("Gaps:\n\n")
				for _, g := range j.Gaps {
					sb.WriteString("- ")
					sb.WriteString(g)
					sb.WriteString("\n")
				}
				sb.WriteString("\n")
			}
		}
	}

	if len(r.Violations) > 0 {
		sb.WriteString("## Violations\n\n")
		for _, v := range r.Violations {
			sb.WriteString("- [")
			sb.WriteString(v.Severity)
			sb.WriteString("] ")
			sb.WriteString(v.Message)
			sb.WriteString(" (")
			sb.WriteString(v.File)
			sb.WriteString(")\n")
		}
	}

	return sb.String()
}
