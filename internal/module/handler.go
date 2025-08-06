package module

import (
	"net/http"
	"time"

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
		TimeDelay int `json:"time_delay" binding:"required"`

		Repeat          int                              `json:"repeat" binding:"required"`
		ActivityRequest []telegrammodule.ActivityRequest `json:"activity_request" binding:"required"` // каждый элемент содержит url и произвольное request_body
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	telegrammodule.ModF_DispatcherActivity(time.Duration(req.TimeDelay)*time.Second, req.Repeat, req.ActivityRequest)

	c.JSON(http.StatusOK, gin.H{"status": "completed"})
}
