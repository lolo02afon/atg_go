package invite_activities

import (
	"atg_go/models"
	"atg_go/pkg/storage"
)

// SendComment публикует комментарий к посту в канале.
// Делегирует выполнение внутренней функции sendComment.
func SendComment(db *storage.DB, accountID int, phone, channelURL string, apiID int, apiHash string, postsCount int, canSend func(channelID, messageID int) (bool, error), userIDs []int, proxy *models.Proxy) (int, int, error) {
	return sendComment(db, accountID, phone, channelURL, apiID, apiHash, postsCount, canSend, userIDs, proxy)
}

// SendReaction устанавливает реакцию на комментарий в обсуждениях канала.
// Внутри вызывает sendReaction.
func SendReaction(db *storage.DB, accountID int, phone, channelURL string, apiID int, apiHash string, msgCount int, proxy *models.Proxy) (int, int, error) {
	return sendReaction(db, accountID, phone, channelURL, apiID, apiHash, msgCount, proxy)
}
