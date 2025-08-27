package storage

import (
	"atg_go/models"
	"log"
)

// CreateChannelPostFact создаёт запись фактических просмотров с нулевыми значениями.
func (db *DB) CreateChannelPostFact(f models.ChannelPostFact) error {
	_, err := db.Conn.Exec(`
                INSERT INTO channel_post_fact (
                        channel_post_theory_id
                ) VALUES ($1)`,
		f.ChannelPostTheoryID,
	)
	if err != nil {
		log.Printf("[DB ERROR] сохранение фактических просмотров: %v", err)
	}
	return err
}
