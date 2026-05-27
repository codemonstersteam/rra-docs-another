package readability

import iodep "github.com/codemonstersteam/rra-docs-another/internal/io"

// Deps — зависимости слайса readability (автономные I/O-объекты).
// Голова знает только их API, не сырые зависимости.
type Deps struct {
	Store iodep.RepoStore
}

// NewDeps собирает зависимости слайса readability для подключения в CLI-роутере.
func NewDeps() Deps {
	return Deps{Store: iodep.NewRepoStore()}
}
