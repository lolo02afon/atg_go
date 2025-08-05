package reaction

import (
	"log"

	"atg_go/pkg/storage"
	"github.com/gin-gonic/gin"
)

// SetupRoutes регистрирует маршруты для работы с реакциями.
func SetupRoutes(r *gin.RouterGroup, db *storage.DB, commentDB *storage.CommentDB) {
	handler := NewHandler(db, commentDB)
	r.POST("/send", handler.SendReaction)
	log.Printf("[ROUTER] Reaction routes registered")
}
