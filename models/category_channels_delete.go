package models

// CategoryChannelsDelete хранит ссылку удалённого канала и причину удаления
// Комментарии пишутся на русском языке по требованию пользователя
// reason должен принимать одно из заранее определённых значений
// "не существует канала по username", "закрытый канал", "недоступно обсуждение"
type CategoryChannelsDelete struct {
	ID         int    `json:"id"`          // Уникальный идентификатор записи
	ChannelURL string `json:"channel_url"` // Ссылка на канал
	Reason     string `json:"reason"`      // Причина удаления
}
