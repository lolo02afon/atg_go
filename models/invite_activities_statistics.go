package models

import "time"

// InviteActivitiesStatistics отражает запись таблицы invite_activities_statistics с агрегированными данными за конкретную дату.
type InviteActivitiesStatistics struct {
	ID              int       `json:"id"`
	Date            time.Time `json:"date"`             // Дата, за которую собрана статистика
	CommentMean     float64   `json:"comment_mean"`     // Среднее количество комментариев на аккаунт
	ReactionMean    float64   `json:"reaction_mean"`    // Среднее количество реакций на аккаунт
	CommentAll      int       `json:"comment_all"`      // Общее количество комментариев за сутки
	ReactionAll     int       `json:"reaction_all"`     // Общее количество реакций за сутки
	AccountFloodBan int       `json:"account_floodban"` // Количество аккаунтов во флуд-бане
	AccountAll      int       `json:"account_all"`      // Всего авторизованных аккаунтов
}
