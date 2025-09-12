package storage

// increaseAccountsSessionsDisconnect увеличивает счётчик в таблице accounts_sessions_disconnect_statistics.
// column принимает название инкрементируемого столбца.
func (db *DB) increaseAccountsSessionsDisconnect(column string) error {
	// Пытаемся обновить существующую запись.
	res, err := db.Conn.Exec(
		"UPDATE accounts_sessions_disconnect_statistics SET " + column + " = " + column + " + 1, date_time = NOW() WHERE id = 1",
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		// Если записи нет, создаём её с начальным значением 1.
		_, err = db.Conn.Exec("INSERT INTO accounts_sessions_disconnect_statistics (id, " + column + ") VALUES (1, 1)")
	}
	return err
}

// IncreaseAccountsCheck увеличивает количество проверенных аккаунтов.
func (db *DB) IncreaseAccountsCheck() error {
	return db.increaseAccountsSessionsDisconnect("accounts_check")
}

// IncreaseAccountsSessionsDisconnect увеличивает количество отключённых сессий.
func (db *DB) IncreaseAccountsSessionsDisconnect() error {
	return db.increaseAccountsSessionsDisconnect("accounts_sessions_disconnect")
}
