package module

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"atg_go/internal/httputil"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// UpdateChannelDuplicateTimes обрабатывает POST /module/channel_duplicate/:id/post_count_day.
// Ожидает JSON-массив времени в формате HH:MM:SS и сохраняет его в БД.
func (h *Handler) UpdateChannelDuplicateTimes(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "некорректный id")
		return
	}

	var times []string
	if err := c.ShouldBindJSON(&times); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "ожидается массив строк HH:MM:SS")
		return
	}

	for _, t := range times {
		if _, err := time.Parse("15:04:05", t); err != nil {
			httputil.RespondError(c, http.StatusBadRequest, "некорректный формат времени: "+t)
			return
		}
	}

	if err := h.DB.UpdateChannelDuplicateTimes(id, pq.StringArray(times)); err != nil {
		log.Printf("[ERROR] обновление post_count_day для channel_duplicate %d: %v", id, err)
		httputil.RespondError(c, http.StatusInternalServerError, "db error")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}
