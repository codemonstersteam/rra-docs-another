package steps

import (
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

var opts = godog.Options{
	Output:    colors.Colored(os.Stdout),
	Format:    "pretty",
	Randomize: -1, // детерминированный порядок
}

func init() {
	godog.BindCommandLineFlags("godog.", &opts)
}

// TestFeatures гоняет все .feature из ../features.
// По умолчанию пропускает @wip (подкоманды до их реализации в S1–S7),
// чтобы main оставался зелёным. GODOG_TAGS переопределяет фильтр.
func TestFeatures(t *testing.T) {
	opts.Paths = []string{"../features"}
	if tags := os.Getenv("GODOG_TAGS"); tags != "" {
		opts.Tags = tags
	} else {
		opts.Tags = "~@wip"
	}

	status := godog.TestSuite{
		Name:                "rra-docs-another",
		ScenarioInitializer: InitializeScenario,
		Options:             &opts,
	}.Run()

	if status != 0 {
		t.Fail()
	}
}

// InitializeScenario регистрирует степы и lifecycle-хуки. World — на сценарий.
func InitializeScenario(ctx *godog.ScenarioContext) {
	w := newWorld()
	ctx.Before(w.beforeScenario)
	w.registerCLISteps(ctx)
}
