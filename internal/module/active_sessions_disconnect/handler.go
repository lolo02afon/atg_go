package active_sessions_disconnect

import (
	"net/http"

	"atg_go/pkg/storage"
	telegrammodule "atg_go/pkg/telegram/module/active_sessions_disconnect"

	"github.com/gin-gonic/gin"
)

// Handler обрабатывает запросы, связанные с активными сессиями.
type Handler struct {
	DB *storage.DB
}

// NewHandler создаёт новый экземпляр обработчика.
func NewHandler(db *storage.DB) *Handler {
	return &Handler{DB: db}
}

// Info вызывает вывод в журнал всех активных сессий аккаунтов.
func (h *Handler) Info(c *gin.Context) {
	if err := telegrammodule.LogAuthorizations(h.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "logged"})
}
