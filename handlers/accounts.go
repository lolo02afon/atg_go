package handlers

import (
	"log"
	"net/http"

	"atg_go/models"
	"atg_go/storage"
	"atg_go/telegram"

	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	DB *storage.DB
}

func NewAccountHandler(db *storage.DB) *AccountHandler {
	return &AccountHandler{DB: db}
}

func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var account models.Account
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	createdAccount, err := h.DB.CreateAccount(account)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании аккаунта"})
		return
	}

	// Запрашиваем код у Telegram асинхронно, чтобы не блокировать ответ
	go func() {
		err := telegram.RequestCode(createdAccount.ApiID, createdAccount.ApiHash, createdAccount.Phone)
		if err != nil {
			log.Printf("Ошибка запроса кода Telegram: %v", err)
		} else {
			log.Printf("Код успешно запрошен для номера: %s", createdAccount.Phone)
		}
	}()

	c.JSON(http.StatusOK, createdAccount)
}
