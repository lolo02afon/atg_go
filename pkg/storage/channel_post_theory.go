package storage

import (
	"atg_go/models"
	"log"
)

// CreateChannelPostTheory сохраняет прогноз распределения просмотров по группам часов
// и возвращает идентификатор созданной записи.
func (db *DB) CreateChannelPostTheory(t models.ChannelPostTheory) (int, error) {
	var id int
	err := db.Conn.QueryRow(`
                INSERT INTO channel_post_theory (
                        channel_post_id,
                        view_7_24hour_theory,
                        view_4_6hour_theory,
                        view_2_3hour_theory,
                        view_1hour_theory,
                        reaction_24hour_theory,
                        repost_24hour_theory
                ) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		t.ChannelPostID,
		t.View724HourTheory,
		t.View46HourTheory,
		t.View23HourTheory,
		t.View1HourTheory,
		t.Reaction24HourTheory,
		t.Repost24HourTheory,
	).Scan(&id)
	if err != nil {
		log.Printf("[DB ERROR] сохранение теории просмотров: %v", err)
	}
	return id, err
}
