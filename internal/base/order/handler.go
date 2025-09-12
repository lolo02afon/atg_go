package order

import (
	"log"
	"strconv"

	"atg_go/internal/technical/httputil"
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
		httputil.RespondError(c, 400, "invalid data")
		return
	}

	created, err := h.DB.CreateOrder(o)
	if err != nil {
		log.Printf("[ERROR] не удалось создать заказ: %v", err)
		httputil.RespondError(c, 500, "db error")
		return
	}

	c.JSON(200, created)
}

// GetCategories возвращает список доступных категорий из таблицы categories
func (h *Handler) GetCategories(c *gin.Context) {
	names, err := h.DB.GetCategoryNames()
	if err != nil {
		httputil.RespondError(c, 500, "db error")
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
		httputil.RespondError(c, 400, "invalid data")
		return
	}
	updated, err := h.DB.UpdateOrderAccountsNumber(id, input.AccountsNumberTheory)
	if err != nil {
		log.Printf("[ERROR] не удалось обновить заказ: %v", err)
		httputil.RespondError(c, 500, "db error")
		return
	}
	c.JSON(200, updated)
}

// DeleteOrder удаляет заказ и освобождает связанные аккаунты
func (h *Handler) DeleteOrder(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.DB.DeleteOrder(id); err != nil {
		log.Printf("[ERROR] не удалось удалить заказ: %v", err)
		httputil.RespondError(c, 500, "db error")
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}
