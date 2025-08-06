package storage

import (
	"database/sql"
	"time"
)

// ActivityTypeReaction — значение поля activity_type для реакций.
const ActivityTypeReaction = "reaction"

// ActivityTypeComment — значение поля activity_type для комментариев.
const ActivityTypeComment = "comment"

// SaveActivity сохраняет действие аккаунта в таблице activity вместе со временем.
func (db *DB) SaveActivity(accountID, channelID, messageID int, activityType string) error {
	_, err := db.Conn.Exec(
		`INSERT INTO activity (id_account, id_channel, id_message, activity_type, date_time) VALUES ($1, $2, $3, $4, $5)`,
		accountID, channelID, messageID, activityType, time.Now(),
	)
	return err
}

// SaveReaction сохраняет информацию о реакции в таблице activity.
func (db *DB) SaveReaction(accountID, channelID, messageID int) error {
	return db.SaveActivity(accountID, channelID, messageID, ActivityTypeReaction)
}

// SaveComment сохраняет информацию о комментарии в таблице activity.
// messageID — идентификатор поста, к которому оставлен комментарий.
func (db *DB) SaveComment(accountID, channelID, messageID int) error {
	return db.SaveActivity(accountID, channelID, messageID, ActivityTypeComment)
}

// HasComment проверяет, существует ли комментарий с указанным идентификатором
// для заданной учетной записи. Возвращает true, если запись с activity_type = 'comment'
// уже есть в таблице.
func (db *DB) HasComment(accountID, messageID int) (bool, error) {
	var exists bool
	err := db.Conn.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM activity WHERE id_account = $1 AND id_message = $2 AND activity_type = 'comment')`,
		accountID, messageID,
	).Scan(&exists)
	return exists, err
}

// HasCommentForPost проверяет, существует ли комментарий с указанным идентификатором
// для заданного канала. Возвращает true при наличии записи с типом 'comment'.
func (db *DB) HasCommentForPost(channelID, messageID int) (bool, error) {
	var exists bool
	err := db.Conn.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM activity WHERE id_channel = $1 AND id_message = $2 AND activity_type = 'comment')`,
		channelID, messageID,
	).Scan(&exists)
	return exists, err
}

// GetLastReactionMessageID возвращает ID сообщения, на которое аккаунт
// поставил реакцию последним в рамках указанного канала.
// Если реакций ещё не было, возвращает 0.
func (db *DB) GetLastReactionMessageID(accountID, channelID int) (int, error) {
	var messageID int
	err := db.Conn.QueryRow(
		`SELECT id_message FROM activity
                 WHERE id_account = $1 AND id_channel = $2 AND activity_type = $3
                 ORDER BY date_time DESC LIMIT 1`,
		accountID, channelID, ActivityTypeReaction,
	).Scan(&messageID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return messageID, nil
}

// CanReactOnMessage проверяет, можно ли аккаунту поставить реакцию на
// сообщение с указанным ID в заданном канале. Разница между ID должна быть
// не менее 10. Если аккаунт ещё не ставил реакций в канале, возвращает true.
func (db *DB) CanReactOnMessage(accountID, channelID, messageID int) (bool, error) {
	lastID, err := db.GetLastReactionMessageID(accountID, channelID)
	if err != nil {
		return false, err
	}
	if lastID == 0 {
		return true, nil
	}
	diff := messageID - lastID
	if diff < 0 {
		diff = -diff
	}
	return diff >= 10, nil
}
