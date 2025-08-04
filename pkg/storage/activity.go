package storage

// SaveActivity persists an account action in the activity table.
func (db *DB) SaveActivity(accountID, channelID, messageID int, activityType string) error {
	_, err := db.Conn.Exec(
		`INSERT INTO activity (id_account, id_channel, id_message, activity_type) VALUES ($1, $2, $3, $4)`,
		accountID, channelID, messageID, activityType,
	)
	return err
}
