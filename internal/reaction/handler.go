package reaction

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

	// Отбрасываем свободные аккаунты, оставляя только привязанные к заказу.
	accounts = common.FilterAccountsWithOrder(accounts)

	// Исключаем аккаунты с включённым мониторингом, чтобы они не тратили лимиты на реакции.
	accounts = common.FilterAccountsWithoutMonitoring(accounts)

	// Унифицированная обработка отсутствия подходящих аккаунтов.
	if len(accounts) == 0 {
		log.Printf("[HANDLER WARN] %s", common.NoOrderedAccountsMessage)
		httputil.RespondError(c, http.StatusNotFound, common.NoOrderedAccountsMessage)
		return
	}

	successCount, errorCount, err := activity.ProcessAccounts(c, accounts, h.CommentDB, func(account models.Account, channelURL string) (bool, error) {
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
			return false, err
		}
		if msgID == 0 {
			log.Printf("[HANDLER INFO] Не найдено подходящих сообщений для аккаунта %s", account.Phone)
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		// Ответ уже отправлен внутри обработчика, поэтому завершаем выполнение.
		return
	}

	// Фиксируем успешные реакции в статистике
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
