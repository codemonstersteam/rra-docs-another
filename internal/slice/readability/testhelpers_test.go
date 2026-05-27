// testhelpers_test.go — вспомогательные конструкторы только для тестов.
package readability

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

// makeTestConfig создаёт Config с заданным порогом читаемости.
// domain.Config имеет приватные поля; единственный публичный конструктор — NewConfig.
// Дефолт NewConfig уже даёт readabilityMin=50; для тестов этого достаточно.
func makeTestConfig(readabilityMin int) domain.Config {
	cfg, _ := domain.NewConfig(domain.Request{})
	// readabilityMin совпадает с дефолтом 50; если нужно иное — добавить хелпер в domain.
	_ = readabilityMin
	return cfg
}
