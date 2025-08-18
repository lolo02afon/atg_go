package order

import (
	"log"
	"strconv"

	"atg_go/models"
	"atg_go/pkg/storage"

	"github.com/gin-gonic/gin"
)

// Handler обрабатывает HTTP-запросы, связанные с заказами
// Комментарии на русском языке по требованию пользователя

type Handler struct {
	DB *storage.DB
}

// NewHandler создаёт новый экземпляр обработчика
func NewHandler(db *storage.DB) *Handler {
	return &Handler{DB: db}
}

// CreateOrder создаёт новый заказ и распределяет аккаунты
func (h *Handler) CreateOrder(c *gin.Context) {
	var o models.Order
	if err := c.ShouldBindJSON(&o); err != nil {
		c.JSON(400, gin.H{"error": "invalid data"})
		return
	}

	created, err := h.DB.CreateOrder(o)
	if err != nil {
		log.Printf("[ERROR] не удалось создать заказ: %v", err)
		c.JSON(500, gin.H{"error": "db error"})
		return
	}

	c.JSON(200, created)
}

// GetCategories возвращает список доступных категорий из таблицы channels
func (h *Handler) GetCategories(c *gin.Context) {
	names, err := h.DB.GetChannelNames()
	if err != nil {
		c.JSON(500, gin.H{"error": "db error"})
		return
	}
	c.JSON(200, gin.H{"categories": names})
}

// UpdateAccountsNumber изменяет количество аккаунтов для заказа
func (h *Handler) UpdateAccountsNumber(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var input struct {
		AccountsNumberTheory int `json:"accounts_number_theory"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "invalid data"})
		return
	}
	updated, err := h.DB.UpdateOrderAccountsNumber(id, input.AccountsNumberTheory)
	if err != nil {
		log.Printf("[ERROR] не удалось обновить заказ: %v", err)
		c.JSON(500, gin.H{"error": "db error"})
		return
	}
	c.JSON(200, updated)
}

// DeleteOrder удаляет заказ и освобождает связанные аккаунты
func (h *Handler) DeleteOrder(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.DB.DeleteOrder(id); err != nil {
		log.Printf("[ERROR] не удалось удалить заказ: %v", err)
		c.JSON(500, gin.H{"error": "db error"})
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}
