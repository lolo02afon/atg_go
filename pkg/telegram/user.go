package telegram

import (
	"atg_go/models"
	"atg_go/pkg/storage"
	userpkg "atg_go/pkg/telegram/user"
)

// GetUserID делегирует получение идентификатора подпакету user.
func GetUserID(db *storage.DB, accountID int, phone string, apiID int, apiHash string, proxy *models.Proxy) (int, error) {
	return userpkg.GetUserID(db, accountID, phone, apiID, apiHash, proxy)
}
