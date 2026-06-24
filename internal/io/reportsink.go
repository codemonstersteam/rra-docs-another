package io

import (
	"fmt"
	"os"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// ReportSink записывает готовый отчёт в stdout или файл. Рендер и выбор
// назначения сделаны до неё (domain.NewReportOutput) — здесь только запись.
type ReportSink struct{}

// NewReportSink создаёт ReportSink.
func NewReportSink() ReportSink { return ReportSink{} }

// Write пишет out.Content() в out.Dest(): File → os.WriteFile, иначе os.Stdout.
// Труба: без ветвлений по данным, кроме маппинга ошибки ФС в ErrReportWrite.
func (s ReportSink) Write(out domain.ReportOutput) error {
	dest := out.Dest()
	if dest.IsFile() {
		if err := os.WriteFile(dest.Path(), []byte(out.Content()+"\n"), 0o644); err != nil {
			return fmt.Errorf("%w: %s: %v", domain.ErrReportWrite, dest.Path(), err)
		}
		return nil
	}
	if _, err := fmt.Fprintln(os.Stdout, out.Content()); err != nil {
		return fmt.Errorf("%w: stdout: %v", domain.ErrReportWrite, err)
	}
	return nil
}
