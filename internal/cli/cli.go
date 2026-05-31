// Package cli — командный роутер rra-docs-another: диспетчеризует первый аргумент в
// подкоманду. Каркас E0 подключает `version`; S1 — `structure`; S2 — `readability`.
// Оставшиеся подкоманды (S3–S7) приходят со своими слайсами.
package cli

import (
	"fmt"
	"io"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
	iodep "github.com/codemonstersteam/rra-docs-another/internal/io"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/drift"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/fitness"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/jtbd"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/readability"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/structure"
)

// Version — версия сборки rra-docs-another. Переопределяется при сборке:
// go build -ldflags "-X github.com/codemonstersteam/rra-docs-another/internal/cli.Version=v1.2.3".
var Version = "0.0.0-dev"

// subcommandsTodo — подкоманды, ещё не реализованные (S4, S7).
var subcommandsTodo = []string{
	"style", "assess",
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
	case "readability":
		return runReadabilityCmd(args[1:], stdout, stderr)
	case "jtbd":
		return runJTBDCmd(args[1:], stdout, stderr)
	case "fitness":
		return runFitnessCmd(args[1:], stdout, stderr)
	case "drift":
		return runDriftCmd(args[1:], stdout, stderr)
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

	deps := structure.NewDeps()
	sink := iodep.NewReportSink()

	report, runErr := structure.ProcessStructure(req, deps)
	return egress(report, runErr, req, sink, stdout)
}

// runReadabilityCmd — точка входа подкоманды readability в CLI-роутере.
func runReadabilityCmd(args []string, stdout, stderr io.Writer) int {
	req, err := readability.ParseArgs(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "rra-docs-another readability: %v\n", err)
		return 2
	}

	deps := readability.NewDeps()
	sink := iodep.NewReportSink()

	report, runErr := readability.ProcessReadability(req, deps)
	return egress(report, runErr, req, sink, stdout)
}

// runJTBDCmd — точка входа подкоманды jtbd в CLI-роутере.
func runJTBDCmd(args []string, stdout, stderr io.Writer) int {
	req, err := jtbd.ParseArgs(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "rra-docs-another jtbd: %v\n", err)
		return 2
	}

	deps := jtbd.NewDeps()
	sink := iodep.NewReportSink()

	report, runErr := jtbd.ProcessJTBD(req, deps)
	return egress(report, runErr, req, sink, stdout)
}

// runFitnessCmd — точка входа подкоманды fitness в CLI-роутере.
func runFitnessCmd(args []string, stdout, stderr io.Writer) int {
	req, err := fitness.ParseArgs(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "rra-docs-another fitness: %v\n", err)
		return 2
	}

	sink := iodep.NewReportSink()

	cfg, cfgErr := domain.NewConfig(req)
	if cfgErr != nil {
		return egress(domain.Report{}, cfgErr, req, sink, stdout)
	}

	// Резолвим и валидируем LLM-подключение здесь (fail-fast по ключу/провайдеру
	// до дорогого I/O); baseURL/model берутся из конфига, клиент их не хардкодит.
	llmCfg, llmErr := domain.NewLLMConfig(req, cfg)
	if llmErr != nil {
		return egress(domain.Report{}, llmErr, req, sink, stdout)
	}

	deps := fitness.NewDeps(cfg, llmCfg)

	report, runErr := fitness.ProcessFitness(req, deps)
	return egress(report, runErr, req, sink, stdout)
}

// runDriftCmd — точка входа подкоманды drift в CLI-роутере.
// Решение --semantic (выбор Judge) принимается здесь, не в голове слайса.
func runDriftCmd(args []string, stdout, stderr io.Writer) int {
	req, err := drift.ParseArgs(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "rra-docs-another drift: %v\n", err)
		return 2
	}

	sink := iodep.NewReportSink()

	// judge по умолчанию — NoopJudge (L6c выключен, ключ не нужен).
	var judge iodep.Judge = iodep.NoopJudge{}
	if req.Semantic {
		cfg, cfgErr := domain.NewConfig(req)
		if cfgErr != nil {
			return egress(domain.Report{}, cfgErr, req, sink, stdout)
		}
		llmCfg, llmErr := domain.NewLLMConfig(req, cfg)
		if llmErr != nil {
			return egress(domain.Report{}, llmErr, req, sink, stdout)
		}
		_ = llmCfg // LLMClient.Judge подключается в S8
	}

	deps := drift.NewDeps(judge)
	report, runErr := drift.ProcessDrift(req, deps)
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
	fmt.Fprintln(w, "  version      показать версию")
	fmt.Fprintln(w, "  structure    L3 структурная полнота")
	fmt.Fprintln(w, "  readability  L1 читаемость")
	fmt.Fprintln(w, "  jtbd         L4 JTBD-присутствие")
	fmt.Fprintln(w, "  fitness      L5 JTBD-пригодность (LLM)")
	fmt.Fprintln(w, "  drift        L6 дрейф документации")
	for _, c := range subcommandsTodo {
		fmt.Fprintf(w, "  %-12s аудит (todo)\n", c)
	}
}
