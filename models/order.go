package models

import (
	"time"

	"github.com/lib/pq"
)

// Order описывает заказ на размещение ссылки в описании аккаунтов
// name - произвольное название заказа
// url_description - ссылка, которая будет вставлена в описание аккаунта
// url_default - ссылка по умолчанию, которая хранится в заказе
// accounts_number_theory - желаемое количество аккаунтов
// accounts_number_fact - количество фактически задействованных аккаунтов
// date_time - время создания заказа
//
// Комментарии в коде на русском языке по требованию пользователя

type Order struct {
	ID                   int            `json:"id"`
	Name                 string         `json:"name"`
	Category             *string        `json:"category"`        // Категория из таблицы channels (может быть NULL)
	URLDescription       string         `json:"url_description"` // Текст ссылки для описания
	URLDefault           string         `json:"url_default"`     // Ссылка по умолчанию
	AccountsNumberTheory int            `json:"accounts_number_theory"`
	AccountsNumberFact   int            `json:"accounts_number_fact"`
	Gender               pq.StringArray `json:"gender"` // Пол(ы) аккаунтов для заказа
	DateTime             time.Time      `json:"date_time"`
}
