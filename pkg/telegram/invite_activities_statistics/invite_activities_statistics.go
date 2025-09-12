package invite_activities_statistics

import (
	"atg_go/models"
	"atg_go/pkg/storage"
	"database/sql"
	"time"
)

// Calculate обновляет средние показатели за текущие сутки и сохраняет их в таблицу invite_activities_statistics.
func Calculate(db *storage.DB) (*models.InviteActivitiesStatistics, error) {
	var stat models.InviteActivitiesStatistics

	// Определяем начало суток по московскому времени
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return nil, err
	}
	dayStart := time.Now().In(loc).Truncate(24 * time.Hour)
	stat.Date = dayStart

	// Получаем существующую запись или создаём новую
	err = db.Conn.QueryRow(
		"SELECT id, comment_all, reaction_all FROM invite_activities_statistics WHERE stat_date = $1",
		dayStart,
	).Scan(&stat.ID, &stat.CommentAll, &stat.ReactionAll)
	if err == sql.ErrNoRows {
		// Создаём запись с нулевыми значениями
		err = db.Conn.QueryRow(
			"INSERT INTO invite_activities_statistics (stat_date, comment_mean, reaction_mean, comment_all, reaction_all, account_floodban, account_all) VALUES ($1, 0, 0, 0, 0, 0, 0) RETURNING id",
			dayStart,
		).Scan(&stat.ID)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// Количество авторизованных аккаунтов
	if err := db.Conn.QueryRow("SELECT COUNT(*) FROM accounts WHERE is_authorized = true").Scan(&stat.AccountAll); err != nil {
		return nil, err
	}

	// Количество аккаунтов во флуд-бане
	if err := db.Conn.QueryRow("SELECT COUNT(*) FROM accounts WHERE floodwait_until IS NOT NULL AND floodwait_until > NOW()").Scan(&stat.AccountFloodBan); err != nil {
		return nil, err
	}

	// Расчёт средних значений
	if stat.AccountAll > 0 {
		stat.CommentMean = float64(stat.CommentAll) / float64(stat.AccountAll)
		stat.ReactionMean = float64(stat.ReactionAll) / float64(stat.AccountAll)
	}

	// Сохраняем обновлённые данные
	_, err = db.Conn.Exec(
		"UPDATE invite_activities_statistics SET comment_mean = $1, reaction_mean = $2, account_floodban = $3, account_all = $4 WHERE stat_date = $5",
		stat.CommentMean, stat.ReactionMean, stat.AccountFloodBan, stat.AccountAll, stat.Date,
	)
	if err != nil {
		return nil, err
	}

	return &stat, nil
}
