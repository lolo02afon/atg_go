package telegram

import (
	"atg_go/models"
	"atg_go/pkg/storage"
	reactionpkg "atg_go/pkg/telegram/reaction"
)

// SendReaction делегирует установку реакции подпакету reaction.
func SendReaction(db *storage.DB, accountID int, phone, channelURL string, apiID int, apiHash string, msgCount int, proxy *models.Proxy) (int, int, error) {
	return reactionpkg.SendReaction(db, accountID, phone, channelURL, apiID, apiHash, msgCount, proxy)
}
