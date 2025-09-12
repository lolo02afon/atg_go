package accounts_auth

import (
	"log"
	"net/http"

	"atg_go/internal/technical/httputil"
	tgauth "atg_go/pkg/telegram/accounts_auth"

	"github.com/gin-gonic/gin"
)

// Check проходит по всем аккаунтам и фиксирует потерю авторизации.
// Возвращает список телефонов, для которых сессия отсутствует.
func (h *AccountHandler) Check(c *gin.Context) {
	accounts, err := h.DB.GetAuthorizedAccounts()
	if err != nil {
		log.Printf("[ACCOUNT AUTH CHECK] ошибка получения аккаунтов: %v", err)
		httputil.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	var unauth []string
	for _, acc := range accounts {
		if !tgauth.Check(h.DB, acc) {
			unauth = append(unauth, acc.Phone)
		}
	}

	if len(unauth) == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "все аккаунты авторизованы"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"неавторизованы": unauth})
}
