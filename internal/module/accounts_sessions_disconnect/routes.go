package accounts_sessions_disconnect

import (
	"atg_go/pkg/storage"

	"github.com/gin-gonic/gin"
)

// SetupRoutes регистрирует маршруты для работы с активными сессиями
// и запускает фоновые отключения.
func SetupRoutes(r *gin.RouterGroup, db *storage.DB) {
	handler := NewHandler(db)
	// Используем POST, чтобы соответствовать соглашению модульных маршрутов.
	r.POST("", handler.Disconnect)
	r.POST("/info", handler.Info)
	// Запускаем фоновые отключения в 02:00 и 11:00 по МСК.
	startBackgroundDisconnect(db)
}
