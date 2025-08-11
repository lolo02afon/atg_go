package comments

import (
	"atg_go/pkg/storage"
	"atg_go/pkg/telegram"
	"database/sql"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type CommentHandler struct {
	DB        *storage.DB
	CommentDB *storage.CommentDB
}

func NewHandler(db *storage.DB, commentDB *storage.CommentDB) *CommentHandler {
	return &CommentHandler{
		DB:        db,
		CommentDB: commentDB,
	}
}

func (h *CommentHandler) SendComment(c *gin.Context) {
	var request struct {
		PostsCount            int   `json:"posts_count" binding:"required"`
		DispatcherActivityMax []int `json:"dispatcher_activity_max" binding:"required"`
		DispatcherPeriod      []int `json:"dispatcher_period" binding:"required"`
	}

	log.Printf("[HANDLER] Starting mass comment request")

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("[HANDLER ERROR] Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}
	if len(request.DispatcherActivityMax) != 2 || len(request.DispatcherPeriod) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dispatcher_activity_max and dispatcher_period must have exactly 2 elements"})
		return
	}

	// Получаем все авторизованные аккаунты
	accounts, err := h.DB.GetAuthorizedAccounts()
	if err != nil {
		log.Printf("[HANDLER ERROR] Account lookup failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get accounts"})
		return
	}
	if len(accounts) == 0 {
		log.Printf("[HANDLER WARN] No authorized accounts found")
		c.JSON(http.StatusNotFound, gin.H{"error": "No authorized accounts available"})
		return
	}

	// Счётчики успешных и неуспешных попыток
	var successCount, errorCount int

	// Инициализируем генератор случайных чисел
	rand.Seed(time.Now().UnixNano())

	// Собираем ID наших аккаунтов в Telegram
	var userIDs []int
	for _, acc := range accounts {
		id, err := telegram.GetUserID(acc.Phone, acc.ApiID, acc.ApiHash, acc.Proxy)
		if err != nil {
			log.Printf("[HANDLER WARN] Не удалось получить ID для %s: %v", acc.Phone, err)
			continue
		}
		userIDs = append(userIDs, id)
	}
	for i, account := range accounts {
		// Задержка между аккаунтами (чтобы не слишком быстро подряд)
		if i > 0 {
			delay := rand.Intn(10) + 6 // 6–15 секунд
			log.Printf("[HANDLER] Аккаунт %s обработан. Ожидание %d секунд перед следующим...", accounts[i-1].Phone, delay)

			for remaining := delay; remaining > 0; remaining -= 5 {
				select {
				case <-c.Request.Context().Done():
					log.Printf("[HANDLER WARN] Request cancelled during delay")
					c.JSON(http.StatusRequestTimeout, gin.H{
						"status":     "Cancelled during delay",
						"processed":  i,
						"successful": successCount,
						"failed":     errorCount,
					})
					return
				default:
				}
				// log.Printf("[HANDLER] %ds remaining...", remaining)
				time.Sleep(5 * time.Second)
			}
		}

		// --- НОВАЯ ЛОГИКА: выбор канала для каждого аккаунта ---
		channelURL, err := h.CommentDB.GetRandomChannel()
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("[HANDLER ERROR] No channels available: %v", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "No channels available"})
			return
		}
		if err != nil {
			log.Printf("[HANDLER ERROR] Channel selection failed for %s: %v", account.Phone, err)
			errorCount++
			continue
		}
		log.Printf("[HANDLER INFO] Selected channel for %s: %s", account.Phone, channelURL)

		// Отправка комментария в выбранный канал
		msgID, _, err := telegram.SendComment(
			h.DB,
			account.ID,
			account.Phone,
			channelURL,
			account.ApiID,
			account.ApiHash,
			request.PostsCount,
			func(channelID, messageID int) (bool, error) {

				exists, err := h.DB.HasCommentForPost(channelID, messageID)
				if err != nil {
					return false, err
				}
				return !exists, nil
			},
			userIDs,
			account.Proxy,
		)
		if err != nil {
			log.Printf("[HANDLER ERROR] Failed for %s: %v", account.Phone, err)
			errorCount++
			continue
		}
		if msgID == 0 {
			log.Printf("[HANDLER INFO] На пост уже оставлен комментарий, пропуск для %s", account.Phone)
			continue
		}

		successCount++
		log.Printf("[HANDLER DEBUG] Success for account: %s", account.Phone)
	}

	// Итоговый ответ (канал убран, т.к. каждый повтор выбирался свой)
	result := gin.H{
		"status":         "Processing complete",
		"total_accounts": len(accounts),
		"successful":     successCount,
		"failed":         errorCount,
	}
	log.Printf("[HANDLER INFO] Final result: %+v", result)
	c.JSON(http.StatusOK, result)
}
