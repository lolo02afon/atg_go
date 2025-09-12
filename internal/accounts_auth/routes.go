package accounts_auth

import (
	"atg_go/pkg/storage"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.RouterGroup, db *storage.DB) {
	handler := NewHandler(db)
	r.POST("/CreateAccount", handler.CreateAccount)
	r.POST("/CreateAccount/verify", handler.VerifyAccount)
}
