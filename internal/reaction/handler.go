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
		MsgCount              int      `json:"msg_count" binding:"required"`
		DispatcherActivityMax []int    `json:"dispatcher_activity_max" binding:"required"`
		DispatcherPeriod      []string `json:"dispatcher_period" binding:"required"`
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

	msk := time.FixedZone("MSK", 3*3600)

	startTime, err1 := time.Parse("15:04", request.DispatcherPeriod[0])
	endTime, err2 := time.Parse("15:04", request.DispatcherPeriod[1])
	if err1 != nil || err2 != nil {
		log.Printf("[HANDLER ERROR] Неверный формат dispatcher_period: %v %v", err1, err2)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid dispatcher_period format"})
		return
	}
	startMin := startTime.Hour()*60 + startTime.Minute()
	endMin := endTime.Hour()*60 + endTime.Minute()

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

		now := time.Now().In(msk)
		current := now.Hour()*60 + now.Minute()

		var outOfRange bool
		if startMin < endMin {
			outOfRange = current < startMin || current >= endMin
		} else {
			outOfRange = current < startMin && current >= endMin
		}

		if outOfRange {
			log.Printf("[HANDLER INFO] Время %s вне диапазона %s-%s МСК, пропуск для %s", now.Format(time.RFC3339), request.DispatcherPeriod[0], request.DispatcherPeriod[1], account.Phone)
			continue
		}

		limit := dailyReactionLimit(account.ID, now, request.DispatcherActivityMax[0], request.DispatcherActivityMax[1])
		count, err := h.DB.CountReactionsForDate(account.ID, now)
		if err != nil {
			log.Printf("[HANDLER ERROR] Не удалось получить количество реакций для %s: %v", account.Phone, err)
			errorCount++
			continue
		}
		if count >= limit {
			log.Printf("[HANDLER INFO] Достигнут дневной лимит реакций (%d/%d) для %s", count, limit, account.Phone)
			continue
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

// dailyReactionLimit вычисляет дневной лимит реакций для аккаунта в заданном диапазоне.
func dailyReactionLimit(accountID int, date time.Time, min, max int) int {
	if max < min {
		max = min
	}
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	seed := int64(accountID) + day.Unix()
	r := rand.New(rand.NewSource(seed))
	return min + r.Intn(max-min+1)
}
