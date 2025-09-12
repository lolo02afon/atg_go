package invite_activities

import (
	"atg_go/internal/technical/activity"
	"atg_go/internal/technical/common"
	"atg_go/internal/technical/httputil"
	"atg_go/models"
	"atg_go/pkg/storage"
	userpkg "atg_go/pkg/telegram/base/user"
	invact "atg_go/pkg/telegram/invite_activities"
	stats "atg_go/pkg/telegram/invite_activities_statistics"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler объединяет обработку комментариев и реакций.
type Handler struct {
	DB        *storage.DB
	CommentDB *storage.CommentDB
}

// NewHandler создаёт обработчик.
func NewHandler(db *storage.DB, commentDB *storage.CommentDB) *Handler {
	return &Handler{DB: db, CommentDB: commentDB}
}

// getOrderedAccounts возвращает аккаунты, закреплённые за заказом и без мониторинга.
func (h *Handler) getOrderedAccounts() ([]models.Account, error) {
	accounts, err := h.DB.GetAuthorizedAccounts()
	if err != nil {
		return nil, err
	}
	accounts = common.FilterAccountsWithOrder(accounts)
	accounts = common.FilterAccountsWithoutMonitoring(accounts)
	return accounts, nil
}

// SendComment публикует комментарии к чужим постам.
func (h *Handler) SendComment(c *gin.Context) {
	var request struct {
		PostsCount int `json:"posts_count" binding:"required"`
	}

	log.Printf("[HANDLER] Запуск массовой отправки комментариев")

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("[HANDLER ERROR] Неверный формат запроса: %v", err)
		httputil.RespondError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	accounts, err := h.getOrderedAccounts()
	if err != nil {
		log.Printf("[HANDLER ERROR] Account lookup failed: %v", err)
		httputil.RespondError(c, http.StatusInternalServerError, "Failed to get accounts")
		return
	}
	if len(accounts) == 0 {
		log.Printf("[HANDLER WARN] %s", common.NoOrderedAccountsMessage)
		httputil.RespondError(c, http.StatusNotFound, common.NoOrderedAccountsMessage)
		return
	}

	var userIDs []int
	for _, acc := range accounts {
		id, err := userpkg.GetUserID(h.DB, acc.ID, acc.Phone, acc.ApiID, acc.ApiHash, acc.Proxy)
		if err != nil {
			log.Printf("[HANDLER WARN] Не удалось получить ID для %s: %v", acc.Phone, err)
			continue
		}
		userIDs = append(userIDs, id)
	}

	successCount, errorCount, err := activity.ProcessAccounts(c, accounts, h.CommentDB, func(account models.Account, channelURL string) (bool, error) {
		msgID, _, err := invact.SendComment(
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
		return
	}

	if err := stats.IncrementComment(h.DB, successCount); err != nil {
		log.Printf("[HANDLER ERROR] не удалось обновить статистику: %v", err)
	}

	result := gin.H{
		"status":         "Processing complete",
		"total_accounts": len(accounts),
		"successful":     successCount,
		"failed":         errorCount,
	}
	log.Printf("[HANDLER INFO] Final result: %+v", result)
	c.JSON(http.StatusOK, result)
}

// SendReaction отправляет реакции к чужим комментариям.
func (h *Handler) SendReaction(c *gin.Context) {
	var request struct {
		MsgCount int `json:"msg_count" binding:"required"`
	}

	log.Printf("[HANDLER] Запуск массовой отправки реакций")

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("[HANDLER ERROR] Неверный формат запроса: %v", err)
		httputil.RespondError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	accounts, err := h.getOrderedAccounts()
	if err != nil {
		log.Printf("[HANDLER ERROR] Ошибка получения аккаунтов: %v", err)
		httputil.RespondError(c, http.StatusInternalServerError, "Failed to get accounts")
		return
	}
	if len(accounts) == 0 {
		log.Printf("[HANDLER WARN] %s", common.NoOrderedAccountsMessage)
		httputil.RespondError(c, http.StatusNotFound, common.NoOrderedAccountsMessage)
		return
	}

	successCount, errorCount, err := activity.ProcessAccounts(c, accounts, h.CommentDB, func(account models.Account, channelURL string) (bool, error) {
		msgID, _, err := invact.SendReaction(
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
			return false, err
		}
		if msgID == 0 {
			log.Printf("[HANDLER INFO] Не найдено подходящих сообщений для аккаунта %s", account.Phone)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return
	}

	if err := stats.IncrementReaction(h.DB, successCount); err != nil {
		log.Printf("[HANDLER ERROR] не удалось обновить статистику: %v", err)
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
