// Package cli — командный роутер rra-docs-another: диспетчеризует первый аргумент в
// подкоманду. E0 (каркас) подключает только `version`; подкоманды-аудиторы
// (structure, readability, jtbd, style, fitness, drift, assess) приходят со
// своими слайсами S1–S7 (см. PLAN.md).
package cli

import (
	"fmt"
	"io"
)

// Version — версия сборки rra-docs-another. Переопределяется при сборке:
// go build -ldflags "-X github.com/codemonstersteam/rra-docs-another/internal/cli.Version=v1.2.3".
var Version = "0.0.0-dev"

// subcommands — подкоманды-аудиторы в порядке слайсов S1–S7. Перечислены в
// usage; до своих тикетов возвращают код 2 («ещё не реализована»).
var subcommands = []string{
	"structure", "readability", "jtbd", "style", "fitness", "drift", "assess",
}

// Run диспетчеризует args (обычно os.Args[1:]) и возвращает код возврата
// процесса: 0 — успех, 2 — ошибка вызова/контракта.
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
	default:
		if isSubcommand(cmd) {
			fmt.Fprintf(stderr, "rra-docs-another: подкоманда %q ещё не реализована (см. PLAN.md)\n", cmd)
			return 2
		}
		fmt.Fprintf(stderr, "rra-docs-another: неизвестная команда %q\n\n", cmd)
		usage(stderr)
		return 2
	}
}

func isSubcommand(name string) bool {
	for _, c := range subcommands {
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
	for _, c := range subcommands {
		fmt.Fprintf(w, "  %-10s аудит (todo)\n", c)
	}
}
