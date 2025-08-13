package module

import (
	"log"
	"net/http"

	"atg_go/pkg/storage"
	telegrammodule "atg_go/pkg/telegram/module"

	"github.com/gin-gonic/gin"
)

// Handler обрабатывает запросы к модулю Telegram.
type Handler struct {
	DB *storage.DB
}

// NewHandler создает новый экземпляр обработчика.
func NewHandler(db *storage.DB) *Handler { return &Handler{DB: db} }

// DispatcherActivity запускает модульную активность диспатчера.
// Ожидает JSON со списком activity_request, где у каждого запроса есть url и request_body.
func (h *Handler) DispatcherActivity(c *gin.Context) {
	var req struct {
		DaysNumber       int                              `json:"days_number" binding:"required"`
		ActivityRequest  []telegrammodule.ActivityRequest `json:"activity_request" binding:"required"`
		ActivityComment  telegrammodule.ActivitySettings  `json:"activity_comment" binding:"required"`
		ActivityReaction telegrammodule.ActivitySettings  `json:"activity_reaction" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Запускаем выполнение активностей в течение заданного количества суток
	telegrammodule.ModF_DispatcherActivity(req.DaysNumber, req.ActivityRequest, req.ActivityComment, req.ActivityReaction)

	c.JSON(http.StatusOK, gin.H{"status": "completed"})
}

// Unsubscribe обрабатывает POST /module/unsubscribe.
//
// Запрос (JSON):
//
//	{
//	  "delay": [min, max],               // массив из двух чисел с диапазоном задержки в секундах
//	  "number_channels_or_groups": N     // количество каналов/групп (>= 0)
//	}
//
// Ответ (200, JSON):
// { "status": "completed" }
//
// Возможные ошибки:
// - 400: неверный формат запроса
// - 400: delay должен содержать два значения
// - 400: number_channels_or_groups должен быть неотрицательным числом
// - 500: внутренняя ошибка отписки
func (h *Handler) Unsubscribe(c *gin.Context) {
	var req struct {
		Delay                  []int `json:"delay" binding:"required"`
		NumberChannelsOrGroups int   `json:"number_channels_or_groups" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат запроса"})
		return
	}

	if len(req.Delay) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "delay должен содержать два значения"})
		return
	}

	if req.NumberChannelsOrGroups < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "number_channels_or_groups должен быть неотрицательным числом"})
		return
	}

	delayRange := [2]int{req.Delay[0], req.Delay[1]}
	log.Printf("[UNSUBSCRIBE] запрос: delay=%v, count=%d", delayRange, req.NumberChannelsOrGroups)

	if err := telegrammodule.ModF_UnsubscribeAll(h.DB, delayRange, req.NumberChannelsOrGroups); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "completed"})
}
