package common

import "atg_go/models"

// NoOrderedAccountsMessage используется в обработчиках, чтобы единообразно
// сообщать об отсутствии авторизованных аккаунтов, привязанных к заказу.
const NoOrderedAccountsMessage = "No authorized ordered accounts available"

// FilterAccountsWithOrder оставляет только те аккаунты, которые прикреплены к заказу.
// Так мы исключаем «свободные» аккаунты из активностей.
func FilterAccountsWithOrder(accounts []models.Account) []models.Account {
	filtered := make([]models.Account, 0, len(accounts))
	for _, acc := range accounts {
		if acc.OrderID != nil {
			filtered = append(filtered, acc)
		}
	}
	return filtered
}

// FilterAccountsWithoutMonitoring исключает аккаунты, для которых включён мониторинг.
// Такие аккаунты выполняют только вспомогательные задачи и не должны участвовать в активностях.
func FilterAccountsWithoutMonitoring(accounts []models.Account) []models.Account {
	filtered := make([]models.Account, 0, len(accounts))
	for _, acc := range accounts {
		if !acc.AccountMonitoring {
			filtered = append(filtered, acc)
		}
	}
	return filtered
}
