package models

// ChannelDuplicate описывает канал-источник, контент которого нужно дублировать
// order_id - идентификатор заказа; при удалении заказа запись удаляется каскадно
// url_channel_duplicate - ссылка на канал-источник
// channel_duplicate_tgid - ID телеграм-канала источника
// post_text_remove - текст, который нужно удалить из поста
// post_text_add - текст, который нужно добавить в конец поста
//
// Комментарии в коде на русском языке по требованию пользователя

type ChannelDuplicate struct {
	ID                   int     `json:"id"`
	OrderID              int     `json:"order_id"`               // ID связанного заказа
	URLChannelDuplicate  string  `json:"url_channel_duplicate"`  // Ссылка на канал-источник
	ChannelDuplicateTGID *string `json:"channel_duplicate_tgid"` // ID телеграм-канала источника
	PostTextRemove       *string `json:"post_text_remove"`       // Текст для удаления
	PostTextAdd          *string `json:"post_text_add"`          // Текст для добавления
}
