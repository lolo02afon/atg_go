package invite_activities_statistics

import (
	"atg_go/pkg/storage"
	"database/sql"
	"time"
)

// IsChannelsLimitActive проверяет, запрещены ли новые подписки для аккаунта.
func IsChannelsLimitActive(db *storage.DB, accountID int) (bool, error) {
	var until sql.NullTime
	err := db.Conn.QueryRow("SELECT channels_limit_until FROM accounts WHERE id = $1", accountID).Scan(&until)
	if err != nil {
		return false, err
	}
	if !until.Valid {
		return false, nil
	}
	return time.Now().Before(until.Time), nil
}

// MarkChannelsLimit устанавливает время, до которого запрещены новые подписки.
func MarkChannelsLimit(db *storage.DB, accountID int, until time.Time) error {
	_, err := db.Conn.Exec("UPDATE accounts SET channels_limit_until = $1 WHERE id = $2", until, accountID)
	return err
}
