package models

import "time"

// AccountsSessionsDisconnect фиксирует статистику отключённых сессий аккаунтов.
type AccountsSessionsDisconnect struct {
	ID                         int       `json:"id"`                           // Уникальный идентификатор записи
	DateTime                   time.Time `json:"date_time"`                    // Дата и время события
	AccountsCheck              int       `json:"accounts_check"`               // Проверенные аккаунты
	AccountsSessionsDisconnect int       `json:"accounts_sessions_disconnect"` // Количество отключённых сессий
}
