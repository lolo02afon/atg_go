package invite_activities

import (
	"atg_go/pkg/storage"
	"github.com/gin-gonic/gin"
)

// SetupRoutes регистрирует маршруты для работы с комментариями и реакциями.
func SetupRoutes(r *gin.RouterGroup, db *storage.DB, commentDB *storage.CommentDB) {
	handler := NewHandler(db, commentDB)
	r.POST("/comment/send", handler.SendComment)
	r.POST("/reaction/send", handler.SendReaction)
}
