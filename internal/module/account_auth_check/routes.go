package account_auth_check

import (
	"atg_go/pkg/storage"

	"github.com/gin-gonic/gin"
)

// SetupRoutes регистрирует маршрут проверки авторизации аккаунтов.
func SetupRoutes(r *gin.RouterGroup, db *storage.DB) {
	handler := NewHandler(db)
	// Используем POST, чтобы не кэшировать результат и соответствовать стилю API.
	r.POST("", handler.Check)
}
