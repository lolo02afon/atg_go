package comments

import (
	"atg_go/pkg/storage"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.RouterGroup, db *storage.DB, commentDB *storage.CommentDB) {
	handler := NewHandler(db, commentDB)
	r.POST("/send", handler.SendComment)

}
