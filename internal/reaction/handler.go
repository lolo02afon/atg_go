package reaction

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

// ReactionHandler отвечает за обработку запросов на массовое добавление реакций.
type ReactionHandler struct {
	DB        *storage.DB
	CommentDB *storage.CommentDB
}

// NewHandler создаёт новый экземпляр обработчика реакций.
func NewHandler(db *storage.DB, commentDB *storage.CommentDB) *ReactionHandler {
	return &ReactionHandler{
		DB:        db,
		CommentDB: commentDB,
	}
}

// SendReaction добавляет реакции к сообщениям обсуждений во всех каналах.
func (h *ReactionHandler) SendReaction(c *gin.Context) {
	var request struct {
		MsgCount              int   `json:"msg_count" binding:"required"`
		DispatcherActivityMax []int `json:"dispatcher_activity_max" binding:"required"`
		DispatcherPeriod      []int `json:"dispatcher_period" binding:"required"`
	}

	log.Printf("[HANDLER] Запуск массовой отправки реакций")

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("[HANDLER ERROR] Неверный формат запроса: %v", err)
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
		log.Printf("[HANDLER ERROR] Ошибка получения аккаунтов: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get accounts"})
		return
	}
	if len(accounts) == 0 {
		log.Printf("[HANDLER WARN] Нет авторизованных аккаунтов")
		c.JSON(http.StatusNotFound, gin.H{"error": "No authorized accounts available"})
		return
	}

	// Счётчики
	var successCount, errorCount int

	rand.Seed(time.Now().UnixNano())

	for i, account := range accounts {
		// Задержка между аккаунтами
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
				time.Sleep(5 * time.Second)
			}
		}

		// Выбор случайного канала
		channelURL, err := h.CommentDB.GetRandomChannel()
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("[HANDLER ERROR] Нет доступных каналов: %v", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "No channels available"})
			return
		}
		if err != nil {
			log.Printf("[HANDLER ERROR] Ошибка выбора канала для %s: %v", account.Phone, err)
			errorCount++
			continue
		}
		log.Printf("[HANDLER INFO] Выбран канал для %s: %s", account.Phone, channelURL)

		msgID, _, err := telegram.SendReaction(
			h.DB,
			account.ID,
			account.Phone,
			channelURL,
			account.ApiID,
			account.ApiHash,
			request.MsgCount,
			account.Proxy,
		)

		if err != nil {
			log.Printf("[HANDLER ERROR] Ошибка отправки реакции для %s: %v", account.Phone, err)
			errorCount++
			continue
		}
		if msgID == 0 {
			log.Printf("[HANDLER INFO] Не найдено подходящих сообщений для аккаунта %s", account.Phone)
			continue
		}

		successCount++
		log.Printf("[HANDLER DEBUG] Успех для аккаунта: %s", account.Phone)
	}

	result := gin.H{
		"status":         "Processing complete",
		"total_accounts": len(accounts),
		"successful":     successCount,
		"failed":         errorCount,
	}
	log.Printf("[HANDLER INFO] Итог: %+v", result)
	c.JSON(http.StatusOK, result)
}
