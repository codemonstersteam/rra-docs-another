package drift

import (
	"github.com/codemonstersteam/rra-docs-another/internal/domain"
	iodep "github.com/codemonstersteam/rra-docs-another/internal/io"
)

// Evaluate — экспортная точка входа L6 для S7 assess.
// Принимает уже прочитанную структуру репо, конфиг и судью (NoopJudge или LLMClient).
// Возвращает LayerOutcome (L6a + опциональный L6c через judge).
func Evaluate(s domain.RepoStructure, cfg domain.Config, judge iodep.Judge) (domain.LayerOutcome, error) {
	claims := extractClaims(s)
	check := NewDriftCheck(s, claims)

	l6aFindings := verifyClaims(check)

	var semFindings []DriftFinding
	if judge.Enabled() {
		promptSet := buildClaimPromptSet(check, cfg)
		verdicts, err := judge.Judge(promptSet)
		if err != nil {
			return domain.LayerOutcome{}, err
		}
		semFindings = mergeSemanticFindings(verdicts)
	}

	report := NewDriftReport(l6aFindings, semFindings)
	return buildDriftOutcome(report), nil
}
