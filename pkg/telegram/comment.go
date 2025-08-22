package telegram

import (
	"atg_go/models"
	"atg_go/pkg/storage"
	commentpkg "atg_go/pkg/telegram/comment"
)

// SendComment делегирует отправку комментария подпакету comment.
func SendComment(db *storage.DB, accountID int, phone, channelURL string, apiID int, apiHash string, postsCount int, canSend func(channelID, messageID int) (bool, error), userIDs []int, proxy *models.Proxy) (int, int, error) {
	return commentpkg.SendComment(db, accountID, phone, channelURL, apiID, apiHash, postsCount, canSend, userIDs, proxy)
}
