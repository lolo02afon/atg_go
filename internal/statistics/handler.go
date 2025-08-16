package statistics

import (
	"atg_go/pkg/storage"
	stats "atg_go/pkg/telegram/statistics"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler обслуживает HTTP-запросы, связанные со статистикой.
type Handler struct {
	DB *storage.DB
}

// NewHandler создаёт новый обработчик статистики.
func NewHandler(db *storage.DB) *Handler {
	return &Handler{DB: db}
}

// Collect рассчитывает показатели и сохраняет их в таблицу statistics.
func (h *Handler) Collect(c *gin.Context) {
	stat, err := stats.Calculate(h.DB)
	if err != nil {
		log.Printf("[HANDLER ERROR] не удалось посчитать статистику: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось посчитать статистику"})
		return
	}
	c.JSON(http.StatusOK, stat)
}
