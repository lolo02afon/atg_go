package module

import (
	"context"
	"net/http"

	"atg_go/internal/httputil"
	telegrammodule "atg_go/pkg/telegram/module"

	"github.com/gin-gonic/gin"
)

// dispatcher.go содержит обработчики запуска и остановки задач диспетчера.
// Вынос этих функций упрощает поддержку и разгружает основной файл обработчика.

// DispatcherActivity запускает модульную активность диспатчера.
// Помимо activity_request ожидает параметры запуска для разных типов действий.
func (h *Handler) DispatcherActivity(c *gin.Context) {
	var req struct {
		DaysNumber                       int                                             `json:"days_number" binding:"required"`
		ActivityRequest                  []telegrammodule.ActivityRequest                `json:"activity_request" binding:"required"`
		ActivityComment                  telegrammodule.ActivitySettings                 `json:"activity_comment" binding:"required"`
		ActivityReaction                 telegrammodule.ActivitySettings                 `json:"activity_reaction" binding:"required"`
		ActivityAccountsUnsubscribe      telegrammodule.UnsubscribeSettings              `json:"activity_accounts_unsubscribe" binding:"required"`
		ActivityActiveSessionsDisconnect telegrammodule.ActiveSessionsDisconnectSettings `json:"activity_active_sessions_disconnect" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Запускаем задачу в отдельной горутине с возможностью отмены.
	ctx, cancel := context.WithCancel(context.Background())

	h.mu.Lock()
	id := h.next
	h.next++
	h.tasks[id] = cancel
	h.mu.Unlock()

	go func(taskID int) {
		defer func() {
			h.mu.Lock()
			delete(h.tasks, taskID)
			h.mu.Unlock()
		}()
		telegrammodule.ModF_DispatcherActivity(ctx, req.DaysNumber, req.ActivityRequest, req.ActivityComment, req.ActivityReaction, req.ActivityAccountsUnsubscribe, req.ActivityActiveSessionsDisconnect)
	}(id)

	c.JSON(http.StatusOK, gin.H{"status": "запущено", "task_id": id})
}

// CancelAllDispatcherActivity отменяет все активные задачи DispatcherActivity.
func (h *Handler) CancelAllDispatcherActivity(c *gin.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, cancel := range h.tasks {
		cancel()
		delete(h.tasks, id)
	}

	c.JSON(http.StatusOK, gin.H{"status": "все задачи остановлены"})
}
