package models

import "time"

// ChannelPost фиксирует публикацию из канала в рамках конкретного заказа
// order_id обязателен: без него невозможно понять, к какому заказу относится пост
// post_date_time хранит время по МСК, чтобы вести хронологию размещений
// post_url сохраняется для быстрого перехода к публикации
// Комментарии в коде на русском языке по требованию пользователя

type ChannelPost struct {
	ID           int       `json:"id"`
	OrderID      int       `json:"order_id"`
	PostDateTime time.Time `json:"post_date_time"`
	PostURL      string    `json:"post_url"`
	// SubsActiveView отражает требуемое число просмотров поста; nil если заказ не ставит цель
	SubsActiveView *int `json:"subs_active_view"`
	// SubsActiveReaction хранит целевое количество реакций; nil когда заказ не задаёт метрику
	SubsActiveReaction *int `json:"subs_active_reaction"`
	// SubsActiveRepost фиксирует требуемые репосты; nil при отсутствии требования
	SubsActiveRepost *int `json:"subs_active_repost"`
}
