package models

import "time"

// Активность представляет собой действие учетной записи, такое как комментарий или реакция.
type Activity struct {
	ID           int       `json:"id"`
	AccountID    int       `json:"id_account"`
	ChannelID    string    `json:"id_channel"`
	MessageID    string    `json:"id_message"`
	ActivityType string    `json:"activity_type"`
	DateTime     time.Time `json:"date_time"`
}
