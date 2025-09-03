package storage

// SaveCategoryChannelDelete сохраняет ссылку канала и причину его недоступности.
func (db *DB) SaveCategoryChannelDelete(url, reason string) error {
	_, err := db.Conn.Exec(
		`INSERT INTO category_channels_delete (channel_url, reason) VALUES ($1, $2)`,
		url, reason,
	)
	return err
}
