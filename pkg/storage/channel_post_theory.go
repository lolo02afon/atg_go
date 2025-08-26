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
                        view_4group_theory,
                        view_3group_theory,
                        view_2group_theory,
                        view_1group_theory
                ) VALUES ($1, $2, $3, $4, $5)`,
		t.ChannelPostID,
		t.View4GroupTheory,
		t.View3GroupTheory,
		t.View2GroupTheory,
		t.View1GroupTheory,
	)
	if err != nil {
		log.Printf("[DB ERROR] сохранение теории просмотров: %v", err)
	}
	return err
}
