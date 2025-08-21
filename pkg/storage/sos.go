package storage

// SaveSos сохраняет сообщение о критичном событии в таблице "Sos".
// Фиксируем только текст, время добавляет сама БД через DEFAULT NOW().
func (db *DB) SaveSos(msg string) error {
	_, err := db.Conn.Exec(`INSERT INTO "Sos" (msg) VALUES ($1)`, msg)
	return err
}
