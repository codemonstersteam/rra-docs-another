package assess

import (
	iodep "github.com/codemonstersteam/rra-docs-another/internal/io"
)

// Deps — зависимости слайса assess.
// LLMConfig и LLM-клиент строятся в голове по ветке L5 (условный резолв).
type Deps struct {
	Store iodep.RepoStore
	Judge iodep.Judge
}

// NewDeps собирает зависимости слайса assess для CLI-роутера.
// Judge = NoopJudge (L6a, без семантического тира).
func NewDeps() Deps {
	return Deps{
		Store: iodep.NewRepoStore(),
		Judge: iodep.NoopJudge{},
	}
}
