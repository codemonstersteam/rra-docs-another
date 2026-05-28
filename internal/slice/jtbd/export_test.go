// export_test.go экспортирует приватные функции для белого ящика тестов.
package jtbd

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

var ExportMatchHeadings = matchHeadings
var ExportBuildJTBDCard = buildJTBDCard
var ExportNormalizeHeading = normalizeHeading

var ExportSpecMaintainer = specMaintainer
var ExportSpecConsumer = specConsumer
var ExportSpecManager = specManager
var ExportSpecAgent = specAgent

// ExportMakeConfig создаёт Config с дефолтными параметрами для тестов.
func ExportMakeConfig() domain.Config {
	cfg, _ := domain.NewConfig(domain.Request{})
	return cfg
}
