package models

import "time"

// Statistics отражает запись таблицы statistics с агрегированными данными за конкретную дату.
type Statistics struct {
	ID              int       `json:"id"`
	Date            time.Time `json:"date"`             // Дата, за которую собрана статистика
	CommentMean     float64   `json:"comment_mean"`     // Среднее количество комментариев на аккаунт
	ReactionMean    float64   `json:"reaction_mean"`    // Среднее количество реакций на аккаунт
	CommentAll      int       `json:"comment_all"`      // Общее количество комментариев за сутки
	ReactionAll     int       `json:"reaction_all"`     // Общее количество реакций за сутки
	AccountFloodBan int       `json:"account_floodban"` // Количество аккаунтов во флуд-бане
	AccountAll      int       `json:"account_all"`      // Всего авторизованных аккаунтов
}
