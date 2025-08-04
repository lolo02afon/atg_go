package reaction

import (
	"atg_go/pkg/storage"
	"log"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.RouterGroup, db *storage.DB, commentDB *storage.CommentDB) {
	handler := NewHandler(db, commentDB)
	r.POST("/send", handler.SendReaction)
	log.Printf("[ROUTER] Reaction routes registered")
}
