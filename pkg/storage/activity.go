package storage

import (
	"database/sql"
	"strconv"
	"time"
)

// ActivityTypeReaction — значение поля activity_type для реакций.
const ActivityTypeReaction = "reaction"

// ActivityTypeComment — значение поля activity_type для комментариев.
const ActivityTypeComment = "comment"

// SaveActivity сохраняет действие аккаунта в таблице activity вместе со временем.
func (db *DB) SaveActivity(accountID, channelID, messageID int, activityType string) error {
	chID := strconv.FormatInt(int64(channelID), 10) // сохраняем ID как строку
	msgID := strconv.FormatInt(int64(messageID), 10)
	_, err := db.Conn.Exec(
		`INSERT INTO activity (id_account, id_channel, id_message, activity_type, date_time) VALUES ($1, $2, $3, $4, $5)`,
		accountID, chID, msgID, activityType, time.Now(),
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
	msgID := strconv.FormatInt(int64(messageID), 10) // сравниваем по строковому ID
	err := db.Conn.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM activity WHERE id_account = $1 AND id_message = $2 AND activity_type = 'comment')`,
		accountID, msgID,
	).Scan(&exists)
	return exists, err
}

// HasCommentForPost проверяет, существует ли комментарий с указанным идентификатором
// для заданного канала. Возвращает true при наличии записи с типом 'comment'.
func (db *DB) HasCommentForPost(channelID, messageID int) (bool, error) {
	var exists bool
	chID := strconv.FormatInt(int64(channelID), 10)
	msgID := strconv.FormatInt(int64(messageID), 10)
	err := db.Conn.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM activity WHERE id_channel = $1 AND id_message = $2 AND activity_type = 'comment')`,
		chID, msgID,
	).Scan(&exists)
	return exists, err
}

// GetLastReactionMessageID возвращает ID сообщения, на которое аккаунт
// поставил реакцию последним в рамках указанного канала.
// Если реакций ещё не было, возвращает 0.
func (db *DB) GetLastReactionMessageID(accountID, channelID int) (int, error) {
	var messageIDStr string
	chID := strconv.FormatInt(int64(channelID), 10)
	err := db.Conn.QueryRow(
		`SELECT id_message FROM activity
                 WHERE id_account = $1 AND id_channel = $2 AND activity_type = $3
                 ORDER BY date_time DESC LIMIT 1`,
		accountID, chID, ActivityTypeReaction,
	).Scan(&messageIDStr)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	messageID, convErr := strconv.Atoi(messageIDStr)
	if convErr != nil {
		return 0, convErr
	}
	return messageID, nil
}

// GetLastCommentMessageID возвращает числовой ID последнего сообщения,
// к которому аккаунт оставил комментарий в указанном канале.
// Если комментариев ещё не было, возвращается 0 и nil.
// В случае ошибки запроса или если значение id_message невозможно
// преобразовать в целое число, функция возвращает 0 и ошибку.
func (db *DB) GetLastCommentMessageID(accountID, channelID int) (int, error) {
	var messageIDStr string
	chID := strconv.FormatInt(int64(channelID), 10)
	err := db.Conn.QueryRow(
		`SELECT id_message FROM activity
                 WHERE id_account = $1 AND id_channel = $2 AND activity_type = $3
                 ORDER BY date_time DESC LIMIT 1`,
		accountID, chID, ActivityTypeComment,
	).Scan(&messageIDStr)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	messageID, convErr := strconv.Atoi(messageIDStr)
	if convErr != nil {
		return 0, convErr
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
