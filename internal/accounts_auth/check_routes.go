package accounts_auth

import (
	"atg_go/pkg/storage"

	"github.com/gin-gonic/gin"
)

// SetupCheckRoutes регистрирует маршрут проверки авторизации аккаунтов.
func SetupCheckRoutes(r *gin.RouterGroup, db *storage.DB) {
	handler := NewHandler(db)
	// Используем POST, чтобы не кэшировать результат и соответствовать стилю API.
	r.POST("", handler.Check)
}
