package drift

import (
	iodep "github.com/codemonstersteam/rra-docs-another/internal/io"
)

// Deps — зависимости слайса drift (автономные I/O-объекты).
// Judge инжектируется из роутера: NoopJudge (без --semantic) или LLMClient (S8).
type Deps struct {
	Store iodep.RepoStore
	Judge iodep.Judge
}

// NewDeps собирает зависимости слайса drift для CLI-роутера.
func NewDeps(judge iodep.Judge) Deps {
	return Deps{
		Store: iodep.NewRepoStore(),
		Judge: judge,
	}
}
