package telegram

import (
	"atg_go/models"
	"atg_go/pkg/storage"
	authpkg "atg_go/pkg/telegram/auth"
)

// AuthHelper переиспользуется из подпакета auth.
type AuthHelper = authpkg.AuthHelper

// RequestCode отправляет код подтверждения и сохраняет хеш в базе.
func RequestCode(apiID int, apiHash, phone string, proxy *models.Proxy, db *storage.DB, accountID int) (string, error) {
	return authpkg.RequestCode(apiID, apiHash, phone, proxy, db, accountID)
}

// CompleteAuthorization завершает авторизацию аккаунта после получения кода.
func CompleteAuthorization(db *storage.DB, accountID, apiID int, apiHash, phone, code, phoneCodeHash string, proxy *models.Proxy) error {
	return authpkg.CompleteAuthorization(db, accountID, apiID, apiHash, phone, code, phoneCodeHash, proxy)
}
