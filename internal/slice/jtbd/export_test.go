// export_test.go экспортирует приватные функции для белого ящика тестов.
package jtbd

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

var ExportMatchHeadings = matchHeadings
var ExportBuildJTBDCard = buildJTBDCard
var ExportNormalizeHeading = normalizeHeading

// Роли извлекаются из дефолтного конфига (словари больше не хардкодятся в Go).
var (
	ExportSpecMaintainer = consumerByRole("maintainer")
	ExportSpecConsumer   = consumerByRole("consumer")
	ExportSpecManager    = consumerByRole("manager")
	ExportSpecAgent      = consumerByRole("agent")
)

// consumerByRole достаёт JTBDConsumer заданной роли из дефолтного конфига.
func consumerByRole(role string) domain.JTBDConsumer {
	cfg, err := domain.NewConfig(domain.Request{})
	if err != nil {
		panic("export_test: дефолтный конфиг невалиден: " + err.Error())
	}
	for _, c := range cfg.JTBDSpec().Consumers() {
		if c.Role() == role {
			return c
		}
	}
	panic("export_test: роль не найдена в дефолтном конфиге: " + role)
}

// ExportMakeConfig создаёт Config с дефолтными параметрами для тестов.
func ExportMakeConfig() domain.Config {
	cfg, _ := domain.NewConfig(domain.Request{})
	return cfg
}
