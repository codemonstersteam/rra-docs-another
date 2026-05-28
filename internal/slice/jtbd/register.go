package jtbd

import iodep "github.com/codemonstersteam/rra-docs-another/internal/io"

// Deps — зависимости слайса jtbd (автономные I/O-объекты).
type Deps struct {
	Store iodep.RepoStore
}

// NewDeps собирает зависимости слайса jtbd для подключения в CLI-роутере.
func NewDeps() Deps {
	return Deps{Store: iodep.NewRepoStore()}
}
