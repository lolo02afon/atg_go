package module

import (
	"context"
	"log"
	"net/http"
	"sync"

	"atg_go/pkg/storage"
	telegrammodule "atg_go/pkg/telegram/module"

	"github.com/gin-gonic/gin"
)

// Handler обрабатывает запросы к модулю Telegram.
// Handler обрабатывает запросы к модулю Telegram и хранит активные задачи.
type Handler struct {
	DB *storage.DB

	mu    sync.Mutex
	tasks map[int]context.CancelFunc
	next  int
}

// NewHandler создает новый экземпляр обработчика.
func NewHandler(db *storage.DB) *Handler {
	return &Handler{DB: db, tasks: make(map[int]context.CancelFunc)}
}

// DispatcherActivity запускает модульную активность диспатчера.
// Ожидает JSON со списком activity_request, где у каждого запроса есть url и request_body.
func (h *Handler) DispatcherActivity(c *gin.Context) {
	var req struct {
		DaysNumber       int                              `json:"days_number" binding:"required"`
		ActivityRequest  []telegrammodule.ActivityRequest `json:"activity_request" binding:"required"`
		ActivityComment  telegrammodule.ActivitySettings  `json:"activity_comment" binding:"required"`
		ActivityReaction telegrammodule.ActivitySettings  `json:"activity_reaction" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Запускаем задачу в отдельной горутине с возможностью отмены.
	ctx, cancel := context.WithCancel(context.Background())

	h.mu.Lock()
	id := h.next
	h.next++
	h.tasks[id] = cancel
	h.mu.Unlock()

	go func(taskID int) {
		defer func() {
			h.mu.Lock()
			delete(h.tasks, taskID)
			h.mu.Unlock()
		}()
		telegrammodule.ModF_DispatcherActivity(ctx, req.DaysNumber, req.ActivityRequest, req.ActivityComment, req.ActivityReaction)
	}(id)

	c.JSON(http.StatusOK, gin.H{"status": "запущено", "task_id": id})
}

// CancelAllDispatcherActivity отменяет все активные задачи DispatcherActivity.
func (h *Handler) CancelAllDispatcherActivity(c *gin.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, cancel := range h.tasks {
		cancel()
		delete(h.tasks, id)
	}

	c.JSON(http.StatusOK, gin.H{"status": "все задачи остановлены"})
}

// Unsubscribe обрабатывает POST /module/unsubscribe.
//
// Запрос (JSON):
//
//	{
//	  "delay": [min, max],               // массив из двух чисел с диапазоном задержки в секундах
//	  "number_channels_or_groups": N     // количество каналов/групп (>= 0)
//	}
//
// Ответ (200, JSON):
// { "status": "completed" }
//
// Возможные ошибки:
// - 400: неверный формат запроса
// - 400: delay должен содержать два значения
// - 400: number_channels_or_groups должен быть неотрицательным числом
// - 500: внутренняя ошибка отписки
func (h *Handler) Unsubscribe(c *gin.Context) {
	var req struct {
		Delay                  []int `json:"delay" binding:"required"`
		NumberChannelsOrGroups int   `json:"number_channels_or_groups" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат запроса"})
		return
	}

	if len(req.Delay) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "delay должен содержать два значения"})
		return
	}

	if req.NumberChannelsOrGroups < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "number_channels_or_groups должен быть неотрицательным числом"})
		return
	}

	delayRange := [2]int{req.Delay[0], req.Delay[1]}
	log.Printf("[UNSUBSCRIBE] запрос: delay=%v, count=%d", delayRange, req.NumberChannelsOrGroups)

	if err := telegrammodule.ModF_UnsubscribeAll(h.DB, delayRange, req.NumberChannelsOrGroups); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "completed"})
}

// OrderLinkUpdate обрабатывает запрос на обновление ссылок в описании аккаунтов
func (h *Handler) OrderLinkUpdate(c *gin.Context) {
	log.Printf("[HANDLER] старт обновления описаний аккаунтов")
	if err := telegrammodule.Modf_OrderLinkUpdate(h.DB); err != nil {
		log.Printf("[HANDLER ERROR] обновление ссылок: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Printf("[HANDLER] обновление ссылок завершено")
	c.JSON(http.StatusOK, gin.H{"status": "links updated"})
}
