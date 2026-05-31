// Package drift реализует слайс S6 — L6a дрейф документации (без ИИ).
// L6c (--semantic) подключается в S8 инъекцией реального LLMClient вместо NoopJudge.
package drift

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

// Claim — извлечённое проверяемое утверждение.
// Kind: "link" (путь в backtick) | "dependency" (пакет из манифеста).
type Claim struct {
	Kind string
	Text string
	File string
	Line int
}

// DriftFinding — утверждение, не подтверждённое репозиторием.
type DriftFinding struct {
	Claim  Claim
	Reason string
}

// DriftCheck — бандл RepoStructure + []Claim; один data-аргумент для
// verifyClaims и buildClaimPromptSet (правило одного аргумента).
type DriftCheck struct {
	structure domain.RepoStructure
	claims    []Claim
}

// NewDriftCheck создаёт DriftCheck из структуры репо и извлечённых утверждений.
func NewDriftCheck(structure domain.RepoStructure, claims []Claim) DriftCheck {
	return DriftCheck{structure: structure, claims: claims}
}

// DriftReport — итоговый бандл L6a- и L6c-находок.
// Узел-конструктор: два data-аргумента → один объект (санкционированное
// место слияния двух источников, не if).
type DriftReport struct {
	l6a      []DriftFinding
	semantic []DriftFinding
}

// NewDriftReport создаёт DriftReport из L6a- и L6c-находок.
func NewDriftReport(l6a []DriftFinding, semantic []DriftFinding) DriftReport {
	return DriftReport{l6a: l6a, semantic: semantic}
}
