package models

import "time"

// Order описывает заказ на размещение ссылки в описании аккаунтов
// name - произвольное название заказа
// url - ссылка на канал
// accounts_number_theory - желаемое количество аккаунтов
// accounts_number_fact - количество фактически задействованных аккаунтов
// date_time - время создания заказа
//
// Комментарии в коде на русском языке по требованию пользователя

type Order struct {
	ID                   int       `json:"id"`
	Name                 string    `json:"name"`
	URL                  string    `json:"url"`
	URLDefault           string    `json:"url_default"` // ссылку из этого поля нельзя отписывать
	AccountsNumberTheory int       `json:"accounts_number_theory"`
	AccountsNumberFact   int       `json:"accounts_number_fact"`
	DateTime             time.Time `json:"date_time"`
}
