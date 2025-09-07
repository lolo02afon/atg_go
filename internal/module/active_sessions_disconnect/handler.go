package active_sessions_disconnect

import (
	"log"
	"net/http"

	"atg_go/internal/httputil"
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

// Info проверяет, доступны ли все авторизованные аккаунты.
// Возвращает сообщение об успехе или список аккаунтов,
// к которым программа потеряла доступ.
func (h *Handler) Info(c *gin.Context) {
	lost, err := telegrammodule.CheckAccountsState(h.DB)
	if err != nil {
		httputil.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if len(lost) == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "все аккаунты авторизованы"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"потеряны": lost})
}

// Disconnect отключает подозрительные сессии на всех авторизованных аккаунтах.
// Возвращает информацию об отключённых устройствах или сообщение об их отсутствии.
func (h *Handler) Disconnect(c *gin.Context) {
	// Структура для получения задержки из тела запроса.
	// Принимаем массив из двух чисел: минимум и максимум задержки.
	var req struct {
		Delay []int `json:"delay"`
	}

	// Отказ, если тело не соответствует ожидаемому формату, чтобы не обрабатывать некорректные данные.
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "некорректное тело запроса")
		return
	}

	// Проверяем, что указаны ровно два значения задержки.
	if len(req.Delay) != 2 {
		httputil.RespondError(c, http.StatusBadRequest, "delay должен содержать два значения")
		return
	}

	minDelay, maxDelay := req.Delay[0], req.Delay[1]
	if minDelay < 0 || maxDelay < 0 || maxDelay < minDelay {
		httputil.RespondError(c, http.StatusBadRequest, "некорректный диапазон задержки")
		return
	}

	res, err := telegrammodule.DisconnectSuspiciousSessions(h.DB, minDelay, maxDelay)
	if err != nil {
		// Логируем ошибку, чтобы понять, где произошёл сбой
		log.Printf("[ACTIVE SESSIONS DISCONNECT] ошибка выполнения: %v", err)
		httputil.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if len(res) == 0 {
		c.JSON(http.StatusOK, gin.H{"Ответ": "Аккаунты не пытались увести"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"Ответ": res})
}
