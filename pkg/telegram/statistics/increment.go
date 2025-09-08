package statistics

import (
	"atg_go/pkg/storage"
	"time"
)

// IncrementReaction увеличивает счётчик реакций за текущие сутки на указанное количество.
func IncrementReaction(db *storage.DB, count int) error {
	return increment(db, 0, count)
}

// IncrementComment увеличивает счётчик комментариев за текущие сутки на указанное количество.
func IncrementComment(db *storage.DB, count int) error {
	return increment(db, count, 0)
}

// increment выполняет обновление записи statistics, добавляя переданные значения.
func increment(db *storage.DB, commentDelta, reactionDelta int) error {
	if commentDelta == 0 && reactionDelta == 0 {
		return nil
	}

	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return err
	}
	dayStart := time.Now().In(loc).Truncate(24 * time.Hour)

	_, err = db.Conn.Exec(
		`INSERT INTO statistics (stat_date, comment_mean, reaction_mean, comment_all, reaction_all, account_floodban, account_all)
                 VALUES ($1, 0, 0, $2, $3, 0, 0)
                 ON CONFLICT (stat_date) DO UPDATE SET
                    comment_all = statistics.comment_all + EXCLUDED.comment_all,
                    reaction_all = statistics.reaction_all + EXCLUDED.reaction_all`,
		dayStart, commentDelta, reactionDelta,
	)
	return err
}
