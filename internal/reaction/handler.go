package reaction

import (
	"atg_go/internal/common"
	"atg_go/internal/httputil"
	"atg_go/models"
	"atg_go/pkg/storage"
	"atg_go/pkg/telegram"
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
		MsgCount int `json:"msg_count" binding:"required"`
	}

	log.Printf("[HANDLER] Запуск массовой отправки реакций")

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("[HANDLER ERROR] Неверный формат запроса: %v", err)
		httputil.RespondError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Получаем все авторизованные аккаунты
	accounts, err := h.DB.GetAuthorizedAccounts()
	if err != nil {
		log.Printf("[HANDLER ERROR] Ошибка получения аккаунтов: %v", err)
		httputil.RespondError(c, http.StatusInternalServerError, "Failed to get accounts")
		return
	}

	// Фильтруем аккаунты, оставляя только те, что закреплены за заказом.
	// Так мы предотвращаем участие свободных аккаунтов в активности.
	var orderedAccounts []models.Account
	for _, acc := range accounts {
		if acc.OrderID != nil {
			orderedAccounts = append(orderedAccounts, acc)
		}
	}
	accounts = orderedAccounts

	if len(accounts) == 0 {
		log.Printf("[HANDLER WARN] Нет авторизованных аккаунтов с заказом")
		httputil.RespondError(c, http.StatusNotFound, "No authorized ordered accounts available")
		return
	}

	// Счётчики
	var successCount, errorCount int

	rand.Seed(time.Now().UnixNano())

	for i, account := range accounts {
		// Задержка между аккаунтами позволяет распределить активность во времени.
		if i > 0 {
			log.Printf("[HANDLER] Аккаунт %s обработан. Ожидание перед следующим...", accounts[i-1].Phone)
			if err := common.WaitWithCancellation(c.Request.Context(), [2]int{6, 15}); err != nil {
				log.Printf("[HANDLER WARN] Request cancelled during delay: %v", err)
				c.JSON(http.StatusRequestTimeout, gin.H{
					"status":     "Cancelled during delay",
					"processed":  i,
					"successful": successCount,
					"failed":     errorCount,
				})
				return
			}
		}

		// Выбор случайного канала
		// PickRandomChannel унифицирует ошибки и прячет детали получения канала.
		channelURL, err := storage.PickRandomChannel(h.CommentDB, *account.OrderID)
		if err != nil {
			if errors.Is(err, storage.ErrNoChannel) {
				// Нет подходящих каналов — дальнейшая отправка реакций бессмысленна.
				log.Printf("[HANDLER ERROR] Нет доступных каналов: %v", err)
				httputil.RespondError(c, http.StatusNotFound, "No channels available")
				return
			}
			// Остальные ошибки логируем и учитываем в счётчике, чтобы не блокировать остальных.
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
