package storage

import (
	"atg_go/models"
	"log"
)

// CreateChannelPostTheory сохраняет прогноз распределения просмотров по группам часов.
// Используется для фиксации ожидаемых просмотров после публикации поста.
func (db *DB) CreateChannelPostTheory(t models.ChannelPostTheory) error {
	_, err := db.Conn.Exec(`
                INSERT INTO channel_post_theory (
                        channel_post_id,
                       view_7_24hour_theory,
                       view_4_6hour_theory,
                       view_2_3hour_theory,
                       view_1hour_theory
                ) VALUES ($1, $2, $3, $4, $5)`,
		t.ChannelPostID,
		t.View724HourTheory,
		t.View46HourTheory,
		t.View23HourTheory,
		t.View1HourTheory,
	)
	if err != nil {
		log.Printf("[DB ERROR] сохранение теории просмотров: %v", err)
	}
	return err
}
