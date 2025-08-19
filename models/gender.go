package models

import "github.com/lib/pq"

// AllowedGenders содержит список допустимых значений пола
var AllowedGenders = []string{"male", "female", "neutral"}

// DefaultGender используется, если пол не указан или указан неверно
const DefaultGender = "neutral"

// IsValidGender проверяет, что переданное значение входит в список допустимых
func IsValidGender(g string) bool {
	for _, allowed := range AllowedGenders {
		if g == allowed {
			return true
		}
	}
	return false
}

// FilterGenders отбрасывает недопустимые значения и гарантирует минимум одно допустимое значение
// Это защищает от рассинхронизации данных при изменении списка AllowedGenders
func FilterGenders(genders pq.StringArray) pq.StringArray {
	var filtered pq.StringArray
	for _, g := range genders {
		if IsValidGender(g) {
			filtered = append(filtered, g)
		}
	}
	if len(filtered) == 0 {
		// Если все значения оказались недопустимыми, используем значение по умолчанию,
		// чтобы заказ или аккаунт всегда имел хотя бы один пол
		filtered = pq.StringArray{DefaultGender}
	}
	return filtered
}
