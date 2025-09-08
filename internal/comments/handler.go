package comments

import (
	"atg_go/internal/activity"
	"atg_go/internal/common"
	"atg_go/internal/httputil"
	"atg_go/models"
	"atg_go/pkg/storage"
	"atg_go/pkg/telegram"
	stats "atg_go/pkg/telegram/statistics"
	"log"
	"net/http"

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

	// Фильтруем аккаунты, оставляя только закреплённые за заказом.
	accounts = common.FilterAccountsWithOrder(accounts)

	// Исключаем аккаунты, используемые для мониторинга, чтобы не мешать их основной задаче.
	accounts = common.FilterAccountsWithoutMonitoring(accounts)

	// Если ни один аккаунт не найден, возвращаем единообразную ошибку.
	if len(accounts) == 0 {
		log.Printf("[HANDLER WARN] %s", common.NoOrderedAccountsMessage)
		httputil.RespondError(c, http.StatusNotFound, common.NoOrderedAccountsMessage)
		return
	}

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
	successCount, errorCount, err := activity.ProcessAccounts(c, accounts, h.CommentDB, func(account models.Account, channelURL string) (bool, error) {
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
			return false, err
		}
		if msgID == 0 {
			log.Printf("[HANDLER INFO] На пост уже оставлен комментарий, пропуск для %s", account.Phone)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		// Процессор уже отправил ответ клиенту, поэтому просто прекращаем обработку.
		return
	}

	// Фиксируем успешные комментарии в статистике
	if err := stats.IncrementComment(h.DB, successCount); err != nil {
		log.Printf("[HANDLER ERROR] не удалось обновить статистику: %v", err)
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
