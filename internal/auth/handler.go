package auth

import (
	"database/sql"
	"errors"
	"log"

	"atg_go/internal/httputil"
	"atg_go/models"
	"atg_go/pkg/storage"
	telegram "atg_go/pkg/telegram"

	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	DB *storage.DB
}

func NewHandler(db *storage.DB) *AccountHandler {
	return &AccountHandler{DB: db}
}

func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var account models.Account
	if err := c.ShouldBindJSON(&account); err != nil {
		httputil.RespondError(c, 400, "Invalid data")
		return
	}
	// Обнуляем ID, чтобы БД назначила его автоматически
	account.ID = 0

	var proxy *models.Proxy
	if account.ProxyID != nil {
		p, err := h.DB.GetProxyByID(*account.ProxyID)
		if err != nil {
			httputil.RespondError(c, 400, "Proxy not found")
			return
		}
		if p.AccountsCount >= 30 {
			httputil.RespondError(c, 400, "Proxy limit reached")
			return
		}
		proxy = p
	}

	// Проверяем соединение с БД перед созданием аккаунта
	if err := h.DB.Conn.PingContext(c.Request.Context()); err != nil {
		log.Printf("[ERROR] Соединение с БД недоступно: %v", err)
		httputil.RespondError(c, 500, "DB connection error")
		return
	}

	// Сохраняем аккаунт и получаем его ID
	created, err := h.DB.CreateAccount(account)
	if err != nil {
		log.Printf("[ERROR] Не удалось создать аккаунт в БД: %v", err)
		httputil.RespondError(c, 500, "DB error")
		return
	}

	// Отправляем код подтверждения и сохраняем хеш в БД
	if _, err := telegram.RequestCode(account.ApiID, account.ApiHash, account.Phone, proxy, h.DB, created.ID); err != nil {
		log.Printf("[ERROR] Не удалось получить код: %v", err)
		httputil.RespondError(c, 500, "Failed to request code")
		return
	}

	log.Printf("[INFO] Аккаунт сохранён в БД с ID=%d", created.ID)
	// Возвращаем статичное сообщение вместо номера кода
	c.JSON(200, gin.H{"результат": "готово, теперь нужно подтвердить кодом"})
}

func (h *AccountHandler) VerifyAccount(c *gin.Context) {
	var input struct {
		Code string `json:"code"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		httputil.RespondError(c, 400, "Invalid code")
		return
	}

	// Получаем последнюю запись аккаунта и фиксируем первичную ошибку
	account, err := h.DB.GetLastAccount()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Логируем отсутствие данных, чтобы понимать, что таблица пуста
			log.Printf("[WARN] В БД нет аккаунтов: %v", err)
			httputil.RespondError(c, 404, "Account not found")
			return
		}

		// Логируем неожиданную ошибку, чтобы понимать, что сломалось при выборке
		log.Printf("[ERROR] Не удалось получить последний аккаунт: %v", err)
		httputil.RespondError(c, 500, "DB error")
		return
	}

	// Фиксируем ID выбранного аккаунта, чтобы убедиться, что берём именно его
	log.Printf("[INFO] Проверяем аккаунт с ID=%d", account.ID)

	// Если последний аккаунт уже авторизован, сообщаем об этом
	if account.IsAuthorized {
		c.JSON(200, gin.H{"результат": "последний аккаунт уже авторизован"})
		return
	}

	if err := telegram.CompleteAuthorization(
		h.DB,
		account.ID,
		account.ApiID,
		account.ApiHash,
		account.Phone,
		input.Code,
		account.PhoneCodeHash,
		account.Proxy,
	); err != nil {
		httputil.RespondError(c, 400, "Auth failed: "+err.Error())
		return
	}

	// Помечаем аккаунт как авторизованный
	if err := h.DB.MarkAccountAsAuthorized(account.ID); err != nil {
		httputil.RespondError(c, 500, "Failed to mark account as authorized")
		return
	}

	c.JSON(200, gin.H{"status": "Authorized!"})
}
