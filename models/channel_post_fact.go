package models

// ChannelPostFact фиксирует фактические просмотры поста по группам часов.
// channel_post_theory_id связывает запись с прогнозом просмотров,
// view_*_fact отражают реальные значения в указанные промежутки времени.
type ChannelPostFact struct {
	ID                  int     `json:"id"`
	ChannelPostTheoryID int     `json:"channel_post_theory_id"`
	View724HourFact     float64 `json:"view_7_24hour_fact"`   // Часы: 7–24 → 0.5–3.2%
	View46HourFact      float64 `json:"view_4_6hour_fact"`    // Часы: 4–6 → 3.7–6.3%
	View23HourFact      float64 `json:"view_2_3hour_fact"`    // Часы: 2–3 → 6.7–11.0%
	View1HourFact       float64 `json:"view_1hour_fact"`      // Часы: 1 → 20.6–25.7%
	Reaction24HourFact  int     `json:"reaction_24hour_fact"` // Реакции за первые 24 часа
	Repost24HourFact    int     `json:"repost_24hour_fact"`   // Репосты за первые 24 часа
}
