// Package cli — командный роутер rra-docs-another: диспетчеризует первый аргумент в
// подкоманду. Каркас E0 подключает `version`; S1 реализует подкоманду `structure`.
// Оставшиеся подкоманды (S2–S7) приходят со своими слайсами.
package cli

import (
	"fmt"
	"io"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
	iodep "github.com/codemonstersteam/rra-docs-another/internal/io"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/structure"
)

// Version — версия сборки rra-docs-another. Переопределяется при сборке:
// go build -ldflags "-X github.com/codemonstersteam/rra-docs-another/internal/cli.Version=v1.2.3".
var Version = "0.0.0-dev"

// subcommandsTodo — подкоманды, ещё не реализованные (S2–S7).
var subcommandsTodo = []string{
	"readability", "jtbd", "style", "fitness", "drift", "assess",
}

// Run диспетчеризует args (обычно os.Args[1:]) и возвращает код возврата
// процесса: 0 — успех, 1 — blocker, 2 — ошибка вызова/контракта.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch cmd := args[0]; cmd {
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, Version)
		return 0
	case "help", "--help", "-h":
		usage(stdout)
		return 0
	case "structure":
		return runStructureCmd(args[1:], stdout, stderr)
	default:
		if isTodoSubcommand(cmd) {
			fmt.Fprintf(stderr, "rra-docs-another: подкоманда %q ещё не реализована (см. PLAN.md)\n", cmd)
			return 2
		}
		fmt.Fprintf(stderr, "rra-docs-another: неизвестная команда %q\n\n", cmd)
		usage(stderr)
		return 2
	}
}

// runStructureCmd — точка входа подкоманды structure в CLI-роутере.
func runStructureCmd(args []string, stdout, stderr io.Writer) int {
	req, err := structure.ParseArgs(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "rra-docs-another structure: %v\n", err)
		return 2
	}

	deps := structure.Deps{Store: iodep.NewRepoStore()}
	sink := iodep.NewReportSink()

	report, runErr := structure.Run(req, deps)
	return egress(report, runErr, req, sink, stdout)
}

// egress — общий выход: форматирует отчёт (успех или ошибку) и возвращает код.
func egress(report domain.Report, err error, req domain.Request, sink iodep.ReportSink, stdout io.Writer) int {
	if err != nil {
		report = buildErrorReport(req, err)
	}
	// Если out = "-", пишем в переданный stdout (а не os.Stdout).
	writeErr := sink.WriteTo(report, req.Format, stdout)
	if writeErr != nil {
		// Запись в stdout не удалась — молча продолжаем (маловероятно).
		_ = writeErr
	}
	return exitCode(report)
}

func isTodoSubcommand(name string) bool {
	for _, c := range subcommandsTodo {
		if c == name {
			return true
		}
	}
	return false
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "rra-docs-another — универсальный аудитор качества документации репозитория")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Использование:")
	fmt.Fprintln(w, "  rra-docs-another <команда> [флаги]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Команды:")
	fmt.Fprintln(w, "  version    показать версию")
	fmt.Fprintln(w, "  structure  L3 структурная полнота")
	for _, c := range subcommandsTodo {
		fmt.Fprintf(w, "  %-10s аудит (todo)\n", c)
	}
}
