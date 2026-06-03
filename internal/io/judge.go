package io

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

// Judge — интерфейс семантического судьи (L6c).
// Реализации: LLMClient (с флагом --semantic) и NoopJudge (без флага).
type Judge interface {
	Judge(set domain.ClaimPromptSet) ([]domain.Verdict, error)
	// Enabled возвращает false для NoopJudge: сигнал пропустить дорогую подготовку
	// (buildClaimPromptSet), которая бессмысленна, когда L6c выключен.
	Enabled() bool
}

// NoopJudge — null-object: тир L6c выключен, ключ не нужен.
// Используется по умолчанию — при отсутствии флага --semantic.
type NoopJudge struct{}

func (NoopJudge) Judge(_ domain.ClaimPromptSet) ([]domain.Verdict, error) {
	return nil, nil
}

func (NoopJudge) Enabled() bool { return false }
