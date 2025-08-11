package module

import (
	"net/http"

	telegrammodule "atg_go/pkg/telegram/module"

	"github.com/gin-gonic/gin"
)

// Handler обрабатывает запросы к модулю Telegram.
type Handler struct{}

// NewHandler создает новый экземпляр обработчика.
func NewHandler() *Handler { return &Handler{} }

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
