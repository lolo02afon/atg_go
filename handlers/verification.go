package handlers

import (
	"atg_go/models"
	"atg_go/storage"
	"atg_go/telegram"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type VerificationHandler struct {
	DB *storage.DB
}

func NewVerificationHandler(db *storage.DB) *VerificationHandler {
	return &VerificationHandler{DB: db}
}

func (h *VerificationHandler) Create(c *gin.Context) {
	var vcode models.VerificationCode
	if err := c.ShouldBindJSON(&vcode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	created, err := h.DB.CreateVerificationCode(vcode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании"})
		return
	}

	c.JSON(http.StatusOK, created)
}

func (h *VerificationHandler) Update(c *gin.Context) {
	var vcode models.VerificationCode
	if err := c.ShouldBindJSON(&vcode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	err := h.DB.UpdateVerificationCode(vcode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении"})
		return
	}

	if vcode.Send {
		acc, err := h.DB.GetAccountByID(vcode.AccountID)
		if err != nil {
			log.Println("Ошибка получения аккаунта:", err)
			return
		}

		err = telegram.CompleteAuthorization(acc.ApiID, acc.ApiHash, acc.Phone, vcode.Code)
		if err != nil {
			log.Println("Ошибка авторизации Telegram:", err)
			return
		}

		log.Println("Аккаунт авторизован")
	}

	c.JSON(http.StatusOK, gin.H{"status": "Обновлено"})
}
