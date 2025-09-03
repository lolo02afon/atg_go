package models

// CategoryChannelsDelete хранит ссылку удалённого канала и причину удаления.
// reason должен принимать одно из значений констант ниже.
type CategoryChannelsDelete struct {
	ID         int    `json:"id"`          // Уникальный идентификатор записи
	ChannelURL string `json:"channel_url"` // Ссылка на канал
	Reason     string `json:"reason"`      // Причина удаления
}

// Значения поля Reason.
const (
	ReasonChannelMissing   = "не существует канала по ссылке" // Канал не найден по URL
	ReasonChannelClosed    = "канал закрыт"                   // Канал закрыт для просмотра
	ReasonDiscussionClosed = "недоступно обсуждение"          // У канала отключены обсуждения
)
