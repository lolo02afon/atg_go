package reaction

import (
	"atg_go/pkg/storage"
	"database/sql"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ReactionHandler struct {
	DB        *storage.DB
	CommentDB *storage.CommentDB
}

func NewHandler(db *storage.DB, commentDB *storage.CommentDB) *ReactionHandler {
	return &ReactionHandler{
		DB:        db,
		CommentDB: commentDB,
	}
}

func (h *ReactionHandler) SendReaction(c *gin.Context) {
	var request struct {
		PostsCount int `json:"posts_count" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Далее идентично comment/handler.go до вызова Telegram API
	accounts, err := h.DB.GetAuthorizedAccounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get accounts"})
		return
	}

	channelURL, err := h.CommentDB.GetRandomChannel()
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "No channels available"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Channel selection failed"})
		return
	}

	var successCount int
	rand.Seed(time.Now().UnixNano())

	for i, account := range accounts {
		if i > 0 {
			delay := rand.Intn(20) + 6 // (0-20) + 6
			time.Sleep(time.Duration(delay) * time.Second)
		}

		if err != nil {
			log.Printf("Failed for %s: %v", account.Phone, err)
			continue
		}
		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "Reactions processed",
		"total_accounts": len(accounts),
		"successful":     successCount,
		"channel":        channelURL,
	})
}
