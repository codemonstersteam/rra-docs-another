// Command rra-docs-another — универсальный аудитор качества документации
// произвольного git-репозитория. Точка входа тонкая: разбор и диспетчеризация
// живут в internal/cli, чтобы роутер юнит-тестировался без процесса.
package main

import (
	"os"

	"github.com/codemonstersteam/rra-docs-another/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
