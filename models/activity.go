package models

// Активность представляет собой действие учетной записи, такое как комментарий или реакция.
type Activity struct {
	ID           int    `json:"id"`
	AccountID    int    `json:"id_account"`
	ChannelID    int    `json:"id_channel"`
	MessageID    int    `json:"id_message"`
	ActivityType string `json:"activity_type"`
}
