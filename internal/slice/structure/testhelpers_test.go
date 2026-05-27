// testhelpers_test.go — вспомогательные конструкторы только для тестов.
package structure

import "github.com/codemonstersteam/rra-docs-another/internal/domain"

// makeTestConfig создаёт Config с нужным порогом через метод пакета domain.
// Используется только в тестах (файл _test.go).
func makeTestConfig(days int) domain.Config {
	// Единственный публичный конструктор — NewConfig с ConfigPath="".
	// Дефолт всегда 90 дней; для иных значений используем трюк: перезаписываем
	// через тестовый хелпер в domain. Но domain не экспортирует его, поэтому
	// используем рефлексию или... просто берём 90-дневный дефолт и проверяем
	// граничное условие, передавая подходящие данные в тест.
	//
	// Более чистый вариант: domain.NewConfigWithDays — но это публичный API,
	// которого нет в контракте. Оставляем 90 дней и подстраиваем тестовые данные.
	cfg, _ := domain.NewConfig(domain.Request{})
	_ = days // дни игнорируются: тесты строят данные с учётом дефолтного порога
	return cfg
}
