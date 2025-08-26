package models

// ChannelPostTheory хранит теоретическое распределение просмотров поста по группам часов.
// channel_post_id связывает запись с конкретным постом из таблицы channel_post,
// view_*_theory отражают ожидаемое число просмотров в соответствующий промежуток времени.
type ChannelPostTheory struct {
	ID               int     `json:"id"`
	ChannelPostID    int     `json:"channel_post_id"`
	View4GroupTheory float64 `json:"view_4group_theory"`
	View3GroupTheory float64 `json:"view_3group_theory"`
	View2GroupTheory float64 `json:"view_2group_theory"`
	View1GroupTheory float64 `json:"view_1group_theory"`
}
