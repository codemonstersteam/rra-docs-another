// export_test.go экспортирует приватные функции для белого ящика тестов.
package structure

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

var ExportCheckRequiredFiles = checkRequiredFiles
var ExportCheckLinksResolve = checkLinksResolve
var ExportCheckDocDrift = checkDocDrift
var ExportCheckStructure = checkStructure
var ExportBuildReport = buildReport

// ExportMakeConfig создаёт Config с заданным порогом (обход приватных полей).
func ExportMakeConfig(days int) domain.Config {
	return makeTestConfig(days)
}
