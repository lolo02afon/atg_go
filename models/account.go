package models

type Account struct {
	ID           int    `json:"id"`            // уникальный идентификатор (можно будет использовать для БД)
	Phone        string `json:"phone"`         // номер телефона
	ApiID        int    `json:"api_id"`        // Api ID от Telegram
	ApiHash      string `json:"api_hash"`      // Api Hash от Telegram
	IsAuthorized bool   `json:"is_authorized"` // статус авторизации
}
