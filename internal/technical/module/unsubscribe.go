package module

import (
	"log"
	"net/http"

	"atg_go/internal/technical/httputil"
	telegrammodule "atg_go/pkg/telegram/technical"

	"github.com/gin-gonic/gin"
)

// unsubscribe.go отвечает за обработку массовой отписки от каналов и групп.
// Отдельный файл позволяет легче ориентироваться в коде модуля.

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
// { "status": "запущено" }
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
		httputil.RespondError(c, http.StatusBadRequest, "неверный формат запроса")
		return
	}

	if len(req.Delay) != 2 {
		httputil.RespondError(c, http.StatusBadRequest, "delay должен содержать два значения")
		return
	}

	if req.NumberChannelsOrGroups < 0 {
		httputil.RespondError(c, http.StatusBadRequest, "number_channels_or_groups должен быть неотрицательным числом")
		return
	}

	delayRange := [2]int{req.Delay[0], req.Delay[1]}
	log.Printf("[UNSUBSCRIBE] запрос: delay=%v, count=%d", delayRange, req.NumberChannelsOrGroups)

	go func() {
		if err := telegrammodule.ModF_UnsubscribeAll(h.DB, delayRange, req.NumberChannelsOrGroups); err != nil {
			log.Printf("[UNSUBSCRIBE] ошибка: %v", err)
		}
	}()
	c.JSON(http.StatusOK, gin.H{"status": "запущено"})
}
