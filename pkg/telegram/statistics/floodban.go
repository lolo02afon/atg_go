package statistics

import (
	"atg_go/pkg/storage"
	"time"
)

// MarkFloodBan фиксирует время окончания флуд-бана для аккаунта.
func MarkFloodBan(db *storage.DB, accountID int, until time.Time) error {
	_, err := db.Conn.Exec("UPDATE accounts SET floodwait_until = $1 WHERE id = $2", until, accountID)
	return err
}
