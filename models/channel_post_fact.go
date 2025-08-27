package models

// ChannelPostFact фиксирует фактические просмотры поста по группам часов.
// channel_post_theory_id связывает запись с прогнозом просмотров,
// view_*_fact отражают реальные значения в указанные промежутки времени.
type ChannelPostFact struct {
	ID                  int     `json:"id"`
	ChannelPostTheoryID int     `json:"channel_post_theory_id"`
	View4GroupFact      float64 `json:"view_4group_fact"` // Часы: 7–24 → 0.5–3.2%
	View3GroupFact      float64 `json:"view_3group_fact"` // Часы: 4–6 → 3.7–6.3%
	View2GroupFact      float64 `json:"view_2group_fact"` // Часы: 2–3 → 6.7–11.0%
	View1GroupFact      float64 `json:"view_1group_fact"` // Часы: 1 → 20.6–25.7%
}
