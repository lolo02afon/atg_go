package models

// ChannelPostTheory хранит теоретическое распределение просмотров поста по группам часов.
// channel_post_id связывает запись с конкретным постом из таблицы channel_post,
// view_*_theory отражают ожидаемое число просмотров в соответствующий промежуток времени.
type ChannelPostTheory struct {
	ID                int     `json:"id"`
	ChannelPostID     int     `json:"channel_post_id"`
	View724HourTheory float64 `json:"view_7_24hour_theory"` // Часы: 7–24 → 0.5–3.2%
	View46HourTheory  float64 `json:"view_4_6hour_theory"`  // Часы: 4–6 → 3.7–6.3%
	View23HourTheory  float64 `json:"view_2_3hour_theory"`  // Часы: 2–3 → 6.7–11.0%
	View1HourTheory   float64 `json:"view_1hour_theory"`    // Часы: 1 → 20.6–25.7%
}
