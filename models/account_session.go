package models

import "time"

// AccountSession хранит сериализованную сессию Telegram для аккаунта.
type AccountSession struct {
	ID       int       `json:"id"`        // Уникальный идентификатор записи
	DateTime time.Time `json:"date_time"` // Время сохранения сессии
	Account  int       `json:"account"`   // ID аккаунта, которому принадлежит сессия
	DataJSON string    `json:"data_json"` // Содержимое сессии в формате JSON
}
