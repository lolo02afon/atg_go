package models

import "time"

// Sos хранит минимальные сведения о критическом событии.
// Дополнительно сохраняется время создания записи, чтобы можно было отследить момент сбоя.
type Sos struct {
	DateTime time.Time `json:"date_time"`
	Msg      string    `json:"msg"`
}
