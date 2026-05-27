package structure

import (
	"io"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// ParseArgs — публичный ингресс-адаптер (делегирует приватному parseStructureArgs).
func ParseArgs(args []string, stderr io.Writer) (domain.Request, error) {
	return parseStructureArgs(args, stderr)
}

// Run — публичная голова слайса structure (делегирует runStructure).
func Run(req domain.Request, deps Deps) (domain.Report, error) {
	return runStructure(req, deps)
}
