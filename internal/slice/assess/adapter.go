package assess

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// validUpTo — допустимые значения флага --up-to.
var validUpTo = map[string]bool{
	"L1": true, "L2": true, "L3": true,
	"L4": true, "L5": true, "L6": true,
}

// ParseArgs — ингресс-адаптер слайса assess.
// args — os.Args[2:] (после имени подкоманды "assess").
func ParseArgs(args []string, stderr io.Writer) (domain.Request, error) {
	fs := flag.NewFlagSet("assess", flag.ContinueOnError)
	fs.SetOutput(stderr)

	format := fs.String("format", "json", "формат отчёта: md|json")
	out := fs.String("out", "-", "куда писать отчёт; - = stdout")
	configPath := fs.String("config", "", "путь к конфигу")
	upTo := fs.String("up-to", "", "выполнить слои до указанного включительно (L1..L6)")
	llmProvider := fs.String("llm-provider", "", "провайдер LLM")
	llmBaseURL := fs.String("llm-base-url", "", "базовый URL LLM-эндпоинта")
	llmModel := fs.String("llm-model", "", "модель LLM")

	positional, flagArgs := splitPositional(args)
	if err := fs.Parse(flagArgs); err != nil {
		return domain.Request{}, fmt.Errorf("parse args: %w", err)
	}
	positional = append(positional, fs.Args()...)

	if *upTo != "" && !validUpTo[strings.ToUpper(*upTo)] {
		return domain.Request{}, fmt.Errorf("недопустимое значение --up-to: %q (допустимо L1..L6)", *upTo)
	}
	if *upTo != "" {
		*upTo = strings.ToUpper(*upTo)
	}

	path := "."
	if len(positional) > 0 {
		path = positional[0]
	}

	return domain.Request{
		Command:     "assess",
		Path:        path,
		Format:      *format,
		Out:         *out,
		ConfigPath:  *configPath,
		UpTo:        *upTo,
		LLMProvider: *llmProvider,
		LLMBaseURL:  *llmBaseURL,
		LLMModel:    *llmModel,
	}, nil
}

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
