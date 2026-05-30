package fitness

import (
	"github.com/codemonstersteam/rra-docs-another/internal/domain"
	iodep "github.com/codemonstersteam/rra-docs-another/internal/io"
)

// Deps — зависимости слайса fitness (автономные I/O-объекты + конфиг промптов).
type Deps struct {
	Store  iodep.RepoStore
	LLM    LLMClient
	Config domain.Config
}

// NewDeps собирает зависимости слайса fitness для CLI-роутера.
// req содержит параметры LLM-подключения (provider, base-url, model).
func NewDeps(req domain.Request, cfg domain.Config) Deps {
	return Deps{
		Store:  iodep.NewRepoStore(),
		LLM:    NewLLMClient(req.LLMProvider, req.LLMBaseURL, req.LLMModel),
		Config: cfg,
	}
}
