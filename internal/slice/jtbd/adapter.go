package jtbd

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// ParseArgs — ингресс-адаптер слайса jtbd: только парсинг флагов и
// позиционного аргумента. args — os.Args[2:] (после "jtbd").
func ParseArgs(args []string, stderr io.Writer) (domain.Request, error) {
	fs := flag.NewFlagSet("jtbd", flag.ContinueOnError)
	fs.SetOutput(stderr)

	format := fs.String("format", "md", "формат отчёта: md|json")
	out := fs.String("out", "-", "куда писать отчёт; - = stdout")
	configPath := fs.String("config", "", "путь к конфигу")

	positional, flagArgs := splitPositional(args)

	if err := fs.Parse(flagArgs); err != nil {
		return domain.Request{}, fmt.Errorf("parse args: %w", err)
	}

	positional = append(positional, fs.Args()...)

	path := "."
	if len(positional) > 0 {
		path = positional[0]
	}

	return domain.Request{
		Command:    "jtbd",
		Path:       path,
		Format:     *format,
		Out:        *out,
		ConfigPath: *configPath,
	}, nil
}

// splitPositional разделяет args на позиционные (не начинаются с '-') и флаги.
func splitPositional(args []string) (positional, flags []string) {
	i := 0
	for i < len(args) {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, a)
		}
		i++
	}
	return
}
