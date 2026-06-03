package drift

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

var ExportExtractClaims = extractClaims
var ExportVerifyClaims = verifyClaims
var ExportBuildClaimPromptSet = buildClaimPromptSet
var ExportMergeSemanticFindings = mergeSemanticFindings
var ExportBuildDriftOutcome = buildDriftOutcome
var ExportIsFilePath = isFilePath

// ExportDefaultCfg возвращает конфиг с дефолтными значениями (MaxJudgeCalls=20).
func ExportDefaultCfg() domain.Config {
	cfg, _ := domain.NewConfig(domain.Request{})
	return cfg
}
