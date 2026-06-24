package domain

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ── Назначение и готовый к записи отчёт (рефактор по уроку D1) ───────────────
//
// Развилки «куда писать» (req.Out) и «в каком формате» (req.Format) — чистая
// логика над Request, не I/O и не side-инъекция. Собираются в один ReportOutput
// конструктором NewReportOutput; I/O-труба ReportSink.Write получает его готовым.
// См. docs/design/assess/egress.md.

// Destination — куда писать отчёт. Резолвится из req.Out; неэкспортируемые поля.
type Destination struct {
	toFile bool
	path   string
}

// IsFile сообщает, пишем ли в файл (иначе — stdout).
func (d Destination) IsFile() bool { return d.toFile }

// Path — путь файла (валиден только при IsFile).
func (d Destination) Path() string { return d.path }

// resolveDestination — чистая развилка по req.Out: "" / "-" → stdout, иначе файл.
func resolveDestination(req Request) Destination {
	if req.Out == "" || req.Out == "-" {
		return Destination{toFile: false}
	}
	return Destination{toFile: true, path: req.Out}
}

// ReportOutput — отрендеренный отчёт + его назначение, готовый к записи.
// Неэкспортируемые поля: создаётся только через NewReportOutput.
type ReportOutput struct {
	content string
	dest    Destination
}

// Content — сериализованный отчёт.
func (o ReportOutput) Content() string { return o.content }

// Dest — куда его писать.
func (o ReportOutput) Dest() Destination { return o.dest }

// NewReportOutput — конструктор-узел: рендерит отчёт по req.Format и резолвит
// req.Out в Destination, собирая единый ReportOutput.
// Антецедент: req.Format ∈ {"", "json", "md"}. Failure: ErrUnknownFormat.
func NewReportOutput(report Report, req Request) (ReportOutput, error) {
	content, err := renderReport(report, req.Format)
	if err != nil {
		return ReportOutput{}, err
	}
	return ReportOutput{content: content, dest: resolveDestination(req)}, nil
}

// renderReport — чистый хелпер: сериализует отчёт в формате format.
// Failure: ErrUnknownFormat (формат вне {"", "json", "md"}).
func renderReport(report Report, format string) (string, error) {
	switch strings.ToLower(format) {
	case "", "json":
		b, err := json.Marshal(report)
		if err != nil {
			return "", fmt.Errorf("render json: %w", err)
		}
		return string(b), nil
	case "md":
		return renderMarkdown(report), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownFormat, format)
	}
}

// renderMarkdown генерирует человекочитаемый Markdown-отчёт.
func renderMarkdown(r Report) string {
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
