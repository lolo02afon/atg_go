package statistics

import (
	"atg_go/models"
	"atg_go/pkg/storage"
	"time"
)

// Calculate вычисляет статистику по базе и сохраняет её в таблицу statistics.
func Calculate(db *storage.DB) (*models.Statistics, error) {
	var stat models.Statistics

	// Загружаем часовой пояс Москвы
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return nil, err
	}

	// Определяем начало и конец текущих суток по московскому времени
	now := time.Now().In(loc)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	dayEnd := dayStart.Add(24 * time.Hour)
	stat.Date = dayStart

	// Количество авторизованных аккаунтов
	if err := db.Conn.QueryRow("SELECT COUNT(*) FROM accounts WHERE is_authorized = true").Scan(&stat.AccountAll); err != nil {
		return nil, err
	}

	// Общее количество комментариев за текущие сутки
	if err := db.Conn.QueryRow(
		"SELECT COUNT(*) FROM activity WHERE activity_type = 'comment' AND date_time >= $1 AND date_time < $2",
		dayStart.UTC(), dayEnd.UTC(),
	).Scan(&stat.CommentAll); err != nil {
		return nil, err
	}

	// Общее количество реакций за текущие сутки
	if err := db.Conn.QueryRow(
		"SELECT COUNT(*) FROM activity WHERE activity_type = 'reaction' AND date_time >= $1 AND date_time < $2",
		dayStart.UTC(), dayEnd.UTC(),
	).Scan(&stat.ReactionAll); err != nil {
		return nil, err
	}

	// Количество аккаунтов, находящихся во флуд-бане
	if err := db.Conn.QueryRow("SELECT COUNT(*) FROM accounts WHERE floodwait_until IS NOT NULL AND floodwait_until > NOW()").Scan(&stat.AccountFloodBan); err != nil {
		return nil, err
	}

	// Расчёт средних значений
	if stat.AccountAll > 0 {
		stat.CommentMean = float64(stat.CommentAll) / float64(stat.AccountAll)
		stat.ReactionMean = float64(stat.ReactionAll) / float64(stat.AccountAll)
	}

	// Сохраняем или обновляем запись в таблице statistics для текущей даты
	err = db.Conn.QueryRow(
		"INSERT INTO statistics (stat_date, comment_mean, reaction_mean, comment_all, reaction_all, account_floodban, account_all) VALUES ($1, $2, $3, $4, $5, $6, $7) "+
			"ON CONFLICT (stat_date) DO UPDATE SET comment_mean = EXCLUDED.comment_mean, reaction_mean = EXCLUDED.reaction_mean, comment_all = EXCLUDED.comment_all, reaction_all = EXCLUDED.reaction_all, account_floodban = EXCLUDED.account_floodban, account_all = EXCLUDED.account_all "+
			"RETURNING id, stat_date",
		stat.Date, stat.CommentMean, stat.ReactionMean, stat.CommentAll, stat.ReactionAll, stat.AccountFloodBan, stat.AccountAll,
	).Scan(&stat.ID, &stat.Date)
	if err != nil {
		return nil, err
	}

	return &stat, nil
}
