package drift

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

var ExportExtractClaimsWithExts = extractClaims
var ExportVerifyClaims = verifyClaims
var ExportBuildClaimPromptSet = buildClaimPromptSet
var ExportMergeSemanticFindings = mergeSemanticFindings
var ExportBuildDriftOutcome = buildDriftOutcome
var ExportIsFilePath = isFilePath
var ExportIsRepoPath = isRepoPath
var ExportHasAllowedExtension = hasAllowedExtension
var ExportTopLevelSet = topLevelSet

// ExportDefaultCfg возвращает конфиг с дефолтными значениями (MaxJudgeCalls=20).
func ExportDefaultCfg() domain.Config {
	cfg, _ := domain.NewConfig(domain.Request{})
	return cfg
}
