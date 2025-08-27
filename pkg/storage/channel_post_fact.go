package storage

import (
	"fmt"

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

// IncrementChannelPostFact увеличивает счётчик просмотров в заданном столбце на единицу.
// column ожидает одно из имён полей вида view_1hour_fact, view_2_3hour_fact и т.д.
func (db *DB) IncrementChannelPostFact(theoryID int, column string) error {
	query := fmt.Sprintf("UPDATE channel_post_fact SET %s = %s + 1 WHERE channel_post_theory_id = $1", column, column)
	if _, err := db.Conn.Exec(query, theoryID); err != nil {
		log.Printf("[DB ERROR] обновление фактических просмотров: %v", err)
		return err
	}
	return nil
}
