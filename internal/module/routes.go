package module

import (
	authcheck "atg_go/internal/module/account_auth_check"
	accountssessions "atg_go/internal/module/accounts_sessions_disconnect"
	"atg_go/pkg/storage"

	"github.com/gin-gonic/gin"
)

// SetupRoutes регистрирует маршруты модуля.
func SetupRoutes(r *gin.RouterGroup, db *storage.DB) {
	handler := NewHandler(db)
	r.POST("/dispatcher_activity", handler.DispatcherActivity)
	r.POST("/dispatcher_activity/cancel_all", handler.CancelAllDispatcherActivity)
	r.POST("/unsubscribe", handler.Unsubscribe)
	r.POST("/order/link_updat", handler.OrderLinkUpdate)
	r.POST("/channel_duplicate/:id/post_count_day", handler.UpdateChannelDuplicateTimes)
	authcheck.SetupRoutes(r.Group("/account_auth_check"), db)
	accountssessions.SetupRoutes(r.Group("/accounts_sessions_disconnect"), db)
}
