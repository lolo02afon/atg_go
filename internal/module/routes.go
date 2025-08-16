package module

import (
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
}
