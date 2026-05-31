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
// llmCfg — валидированное LLM-подключение (резолвинг baseURL/model/ключа — в
// domain.NewLLMConfig); cfg — проектный конфиг (промпты, docs, операционные пороги).
func NewDeps(cfg domain.Config, llmCfg domain.LLMConfig) Deps {
	return Deps{
		Store:  iodep.NewRepoStore(),
		LLM:    NewLLMClient(llmCfg, cfg.LLMCallDelayMs(), cfg.LLMTokenBudget(), cfg.LLMMaxRetries()),
		Config: cfg,
	}
}
