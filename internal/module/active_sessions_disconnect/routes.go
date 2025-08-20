package active_sessions_disconnect

import (
	"atg_go/pkg/storage"

	"github.com/gin-gonic/gin"
)

// SetupRoutes регистрирует маршруты для работы с активными сессиями.
func SetupRoutes(r *gin.RouterGroup, db *storage.DB) {
	handler := NewHandler(db)
	r.GET("/info", handler.Info)
}
