package module

import "github.com/gin-gonic/gin"

// SetupRoutes регистрирует маршруты модуля.
func SetupRoutes(r *gin.RouterGroup) {
	handler := NewHandler()
	r.POST("/dispatcher_activity", handler.DispatcherActivity)
}
