package comments

import (
	"atg_go/internal/common"
	"atg_go/internal/httputil"
	"atg_go/models"
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
		PostsCount int `json:"posts_count" binding:"required"`
	}

	log.Printf("[HANDLER] Запуск массовой отправки комментариев")

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("[HANDLER ERROR] Неверный формат запроса: %v", err)
		httputil.RespondError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Получаем все авторизованные аккаунты
	accounts, err := h.DB.GetAuthorizedAccounts()

	if err != nil {
		log.Printf("[HANDLER ERROR] Account lookup failed: %v", err)
		httputil.RespondError(c, http.StatusInternalServerError, "Failed to get accounts")
		return
	}

	// Оставляем только аккаунты, которые привязаны к заказу.
	// Это важно, поскольку активность разрешена исключительно для аккаунтов,
	// выполняющих заказ, чтобы избежать действий «свободных» аккаунтов.
	var orderedAccounts []models.Account
	for _, acc := range accounts {
		if acc.OrderID != nil {
			orderedAccounts = append(orderedAccounts, acc)
		}
	}
	accounts = orderedAccounts

	if len(accounts) == 0 {
		log.Printf("[HANDLER WARN] No authorized accounts with order found")
		httputil.RespondError(c, http.StatusNotFound, "No authorized ordered accounts available")
		return
	}

	// Счётчики успешных и неуспешных попыток
	var successCount, errorCount int

	// Инициализируем генератор случайных чисел
	rand.Seed(time.Now().UnixNano())

	// Собираем ID наших аккаунтов в Telegram
	var userIDs []int
	for _, acc := range accounts {
		id, err := telegram.GetUserID(h.DB, acc.ID, acc.Phone, acc.ApiID, acc.ApiHash, acc.Proxy)
		if err != nil {
			log.Printf("[HANDLER WARN] Не удалось получить ID для %s: %v", acc.Phone, err)
			continue
		}
		userIDs = append(userIDs, id)
	}
	for i, account := range accounts {
		// Задержка между аккаунтами нужна, чтобы не создавать всплеск активности.
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

		// --- Выбор канала для каждого аккаунта ---
		// Используем категории заказа, чтобы аккаунт не выходил за рамки своей задачи.
		channelURL, err := h.CommentDB.GetRandomChannel(*account.OrderID)
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("[HANDLER ERROR] No channels available: %v", err)
			httputil.RespondError(c, http.StatusNotFound, "No channels available")
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
