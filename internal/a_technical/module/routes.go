package module

import (
	accauth "atg_go/internal/accounts_auth"
	accsess "atg_go/internal/accounts_sessions_disconnect"
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
	accauth.SetupCheckRoutes(r.Group("/account_auth_check"), db)
	accsess.SetupRoutes(r.Group("/accounts_sessions_disconnect"), db)
}
