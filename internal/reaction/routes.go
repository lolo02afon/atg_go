package reaction

import (
	"atg_go/pkg/storage"
	"github.com/gin-gonic/gin"
)

// SetupRoutes регистрирует маршруты для работы с реакциями.
func SetupRoutes(r *gin.RouterGroup, db *storage.DB, commentDB *storage.CommentDB) {
	handler := NewHandler(db, commentDB)
	r.POST("/send", handler.SendReaction)
}
