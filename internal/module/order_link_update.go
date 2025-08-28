package module

import (
	"log"
	"net/http"

	"atg_go/internal/httputil"
	subactive "atg_go/internal/subs_active"
	subsfact "atg_go/internal/subs_fact"
	telegrammodule "atg_go/pkg/telegram/module"

	"github.com/gin-gonic/gin"
)

// order_link_update.go обновляет ссылки в описании аккаунтов.
// Выделение логики в отдельный файл упрощает поиск обработчиков по задачам.

// OrderLinkUpdate обрабатывает запрос на обновление ссылок в описании аккаунтов.
func (h *Handler) OrderLinkUpdate(c *gin.Context) {
	if err := telegrammodule.Modf_OrderLinkUpdate(h.DB); err != nil {
		log.Printf("[HANDLER ERROR] обновление ссылок: %v", err)
		httputil.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := subsfact.SyncWithSubsActiveCount(h.DB); err != nil {
		log.Printf("[HANDLER ERROR] синхронизация подписок: %v", err)
		httputil.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := subactive.ActivateSubscriptions(h.DB); err != nil {
		log.Printf("[HANDLER ERROR] активные подписки: %v", err)
		httputil.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "links updated and subs active"})
}
