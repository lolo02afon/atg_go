package models

// ChannelPostTheory хранит теоретическое распределение просмотров поста по группам часов.
// channel_post_id связывает запись с конкретным постом из таблицы channel_post,
// view_*_theory отражают ожидаемое число просмотров в соответствующий промежуток времени.
type ChannelPostTheory struct {
	ID            int `json:"id"`
	ChannelPostID int `json:"channel_post_id"`
	// Часы: 7–24 → 31.9–39.4%
	View724HourTheory float64 `json:"view_7_24hour_theory"`
	// Часы: 4–6 → 14.9–19.4%
	View46HourTheory float64 `json:"view_4_6hour_theory"`
	// Часы: 2–3 → 17.2–21.7%
	View23HourTheory float64 `json:"view_2_3hour_theory"`
	// Часы: 1 → 20.6–25.7%
	View1HourTheory      float64 `json:"view_1hour_theory"`
	Reaction24HourTheory int     `json:"reaction_24hour_theory"` // Реакции за первые 24 часа
	Repost24HourTheory   int     `json:"repost_24hour_theory"`   // Репосты за первые 24 часа
}
