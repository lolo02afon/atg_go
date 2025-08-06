package module

import "atg_go/pkg/storage"

// SaveActivity сохраняет любую активность (комментарий или реакцию) в таблице activity.
func SaveActivity(db *storage.DB, accountID, channelID, messageID int, activityType string) error {
	return db.SaveActivity(accountID, channelID, messageID, activityType)
}

// SaveReactionActivity сохраняет реакцию, используя стандартный тип activity_type "reaction".
func SaveReactionActivity(db *storage.DB, accountID, channelID, messageID int) error {
	return db.SaveReaction(accountID, channelID, messageID)
}

// SaveCommentActivity сохраняет комментарий с типом activity_type "comment".
// messageID — ID поста, который был прокомментирован.
func SaveCommentActivity(db *storage.DB, accountID, channelID, messageID int) error {
	return db.SaveComment(accountID, channelID, messageID)
}
