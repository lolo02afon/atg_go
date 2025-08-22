package account_auth_check

import (
	"log"
	"net/http"

	"atg_go/pkg/storage"
	authcheck "atg_go/pkg/telegram/module/account_auth_check"

	"github.com/gin-gonic/gin"
)

// Handler обрабатывает запросы проверки авторизации аккаунтов.
type Handler struct {
	DB *storage.DB
}

// NewHandler создаёт новый экземпляр обработчика.
func NewHandler(db *storage.DB) *Handler {
	return &Handler{DB: db}
}

// Check проходит по всем аккаунтам и фиксирует потерю авторизации.
// Возвращает список телефонов, для которых сессия отсутствует.
func (h *Handler) Check(c *gin.Context) {
	accounts, err := h.DB.GetAuthorizedAccounts()
	if err != nil {
		log.Printf("[ACCOUNT AUTH CHECK] ошибка получения аккаунтов: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var unauth []string
	for _, acc := range accounts {
		if !authcheck.Check(h.DB, acc) {
			unauth = append(unauth, acc.Phone)
		}
	}

	if len(unauth) == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "все аккаунты авторизованы"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"неавторизованы": unauth})
}
