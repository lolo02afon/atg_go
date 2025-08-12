package module

import (
	"net/http"
	"strconv"

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

// Unsubscribe отключает все аккаунты от всех каналов и групп.
func (h *Handler) Unsubscribe(c *gin.Context) {
	delayValues := c.QueryArray("delay")
	if len(delayValues) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "нужно передать два значения параметра delay"})
		return
	}

	var delayRange [2]int
	for i, v := range delayValues {
		d, err := strconv.Atoi(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "delay должен содержать числа"})
			return
		}
		delayRange[i] = d
	}

	if err := telegrammodule.ModF_UnsubscribeAll(h.DB, delayRange); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "completed"})
}
