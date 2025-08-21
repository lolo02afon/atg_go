package active_sessions_disconnect

import (
	"log"
	"net/http"

	"atg_go/pkg/storage"
	telegrammodule "atg_go/pkg/telegram/module/active_sessions_disconnect"

	"github.com/gin-gonic/gin"
)

// Handler обрабатывает запросы, связанные с активными сессиями.
type Handler struct {
	DB *storage.DB
}

// NewHandler создаёт новый экземпляр обработчика.
func NewHandler(db *storage.DB) *Handler {
	return &Handler{DB: db}
}

// Info выводит в журнал активные сессии нескольких случайных
// (до пяти) авторизованных аккаунтов, чтобы контролировать объём
// логов и при этом иметь представление о разных пользователях.
func (h *Handler) Info(c *gin.Context) {
	if err := telegrammodule.LogAuthorizations(h.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "logged"})
}

// Disconnect отключает подозрительные сессии на всех авторизованных аккаунтах.
// Возвращает информацию об отключённых устройствах или сообщение об их отсутствии.
func (h *Handler) Disconnect(c *gin.Context) {
	res, err := telegrammodule.DisconnectSuspiciousSessions(h.DB)
	if err != nil {
		// Логируем ошибку, чтобы понять, где произошёл сбой
		log.Printf("[ACTIVE SESSIONS DISCONNECT] ошибка выполнения: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(res) == 0 {
		c.JSON(http.StatusOK, gin.H{"Ответ": "Аккаунты не пытались увести"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"Ответ": res})
}
