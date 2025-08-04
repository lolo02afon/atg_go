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
func (h *Handler) DispatcherActivity(c *gin.Context) {
	var req struct {
		Time   int `json:"time" binding:"required"`
		Repeat int `json:"repeat" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	telegrammodule.ModF_DispatcherActivity(time.Duration(req.Time)*time.Second, req.Repeat)

	c.JSON(http.StatusOK, gin.H{"status": "completed"})
}
