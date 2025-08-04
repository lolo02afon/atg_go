package storage

// SaveActivity persists an account action in the activity table.
func (db *DB) SaveActivity(accountID, channelID, messageID int, activityType string) error {
	_, err := db.Conn.Exec(
		`INSERT INTO activity (id_account, id_channel, id_message, activity_type) VALUES ($1, $2, $3, $4)`,
		accountID, channelID, messageID, activityType,
	)
	return err
}

// HasComment проверяет, оставляла ли учетная запись комментарий к указанному посту.
// Возвращает true, если запись с activity_type = 'comment' уже существует.
func (db *DB) HasComment(accountID, messageID int) (bool, error) {
	var exists bool
	err := db.Conn.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM activity WHERE id_account = $1 AND id_message = $2 AND activity_type = 'comment')`,
		accountID, messageID,
	).Scan(&exists)
	return exists, err
}

// HasCommentForPost проверяет, оставлялся ли комментарий к посту любым из наших аккаунтов.
// Возвращает true, если в таблице activity есть запись с заданными каналом и постом
// и типом activity_type = 'comment'.
func (db *DB) HasCommentForPost(channelID, messageID int) (bool, error) {
	var exists bool
	err := db.Conn.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM activity WHERE id_channel = $1 AND id_message = $2 AND activity_type = 'comment')`,
		channelID, messageID,
	).Scan(&exists)
	return exists, err
}
