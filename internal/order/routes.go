package order

import (
	"atg_go/pkg/storage"

	"github.com/gin-gonic/gin"
)

// SetupRoutes регистрирует маршруты для работы с заказами
func SetupRoutes(r *gin.RouterGroup, db *storage.DB) {
	h := NewHandler(db)
	r.POST("/CreateOrder", h.CreateOrder)
	r.POST("/UpdateAccounts/:id", h.UpdateAccountsNumber)
	r.DELETE("/DeleteOrder/:id", h.DeleteOrder)
=======
