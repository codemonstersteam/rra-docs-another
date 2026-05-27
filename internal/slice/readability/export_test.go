// export_test.go экспортирует приватные функции для белого ящика тестов.
package readability

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

var ExportFleschKincaid = fleschKincaid
var ExportObornevaRus = obornevaRus
var ExportPickFormula = pickFormula
var ExportScoreReadability = scoreReadability
var ExportBuildReport = buildReport
var ExportCyrillicRatio = cyrillicRatio

// ExportMakeConfig создаёт Config с заданным минимумом читаемости.
func ExportMakeConfig(readabilityMin int) domain.Config {
	return makeTestConfig(readabilityMin)
}
