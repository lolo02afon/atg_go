package models

// ChannelDuplicate описывает канал-источник, контент которого нужно дублировать
// order_id - идентификатор заказа; при удалении заказа запись удаляется каскадно
// url_channel_donor - ссылка на канал-источник
// channel_donor_tgid - ID телеграм-канала источника
// post_text_remove - текст, который нужно удалить из поста
// post_text_add - текст, который нужно добавить в конец поста
//
// Комментарии в коде на русском языке по требованию пользователя

type ChannelDuplicate struct {
	ID               int     `json:"id"`
	OrderID          int     `json:"order_id"`           // ID связанного заказа
	URLChannelDonor  string  `json:"url_channel_donor"`  // Ссылка на канал-источник
	ChannelDonorTGID *string `json:"channel_donor_tgid"` // ID телеграм-канала источника
	PostTextRemove   *string `json:"post_text_remove"`   // Текст для удаления
	PostTextAdd      *string `json:"post_text_add"`      // Текст для добавления
	LastPostID       *int    `json:"last_post_id"`       // ID последнего пересланного поста
}
