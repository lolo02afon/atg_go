package module

import (
	"bytes"
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// константный Bearer-токен для внутренних запросов
const bearerToken = "ZXNzIiwiZXhwIjoxNzUyOTU3OTMyLCJpYXQiOjE3NTI5NTQzMzIsImp0aSI6ImM1ZjY0MjcwMjZjYjY1IiwidXNlcl9pZRcNAW-s02Ayz6A"

// ActivityRequest описывает один запрос активности с адресом и телом
// запроса.
type ActivityRequest struct {
	URL         string         `json:"url"`
	RequestBody map[string]any `json:"request_body"`
}

// ActivitySettings задаёт параметры расписания для активности.
type ActivitySettings struct {
	DispatcherActivityMax []int    `json:"dispatcher_activity_max"`
	DispatcherPeriod      []string `json:"dispatcher_period"`
}

// UnsubscribeSettings описывает параметры запуска отписки аккаунтов.
type UnsubscribeSettings struct {
	DispatcherStart string `json:"dispatcher_start"`
}

// ActiveSessionsDisconnectSettings задаёт время отключения активных сессий.
type ActiveSessionsDisconnectSettings struct {
	DispatcherStart string `json:"dispatcher_start"`
}

// ModF_DispatcherActivity выполняет запросы активности в течение
// заданного количества суток и реагирует на отмену контекста.
func ModF_DispatcherActivity(ctx context.Context, daysNumber int, activities []ActivityRequest, commentCfg, reactionCfg ActivitySettings, unsubscribeCfg UnsubscribeSettings, disconnectCfg ActiveSessionsDisconnectSettings) {
	rand.Seed(time.Now().UnixNano())

	// Загружаем часовую зону Москвы и фиксируем текущее время в ней,
	// чтобы дальнейшие расчёты опирались на МСК
	loc, _ := time.LoadLocation("Europe/Moscow")
	start := time.Now().In(loc)

	for day := 0; day < daysNumber; day++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var wg sync.WaitGroup

		for _, act := range activities {
			// Определяем конфигурацию в зависимости от типа активности
			switch {
			case strings.Contains(act.URL, "comment"):
				cfg := commentCfg
				if len(cfg.DispatcherActivityMax) != 2 || len(cfg.DispatcherPeriod) != 2 {
					continue
				}

				wg.Add(1)
				go func(act ActivityRequest, cfg ActivitySettings, offset int) {
					defer wg.Done()

					// Парсим временные границы выполнения
					startTime, err1 := time.Parse("15:04", cfg.DispatcherPeriod[0])
					endTime, err2 := time.Parse("15:04", cfg.DispatcherPeriod[1])
					if err1 != nil || err2 != nil {
						return
					}
					startMin := startTime.Hour()*60 + startTime.Minute()
					endMin := endTime.Hour()*60 + endTime.Minute()
					minAct := cfg.DispatcherActivityMax[0]
					maxAct := cfg.DispatcherActivityMax[1]
					if endMin <= startMin || maxAct < minAct {
						return
					}

					count := rand.Intn(maxAct-minAct+1) + minAct

					currentDay := start.AddDate(0, 0, offset)
					// Начало окна активности в московском часовом поясе
					windowStart := time.Date(currentDay.Year(), currentDay.Month(), currentDay.Day(), startTime.Hour(), startTime.Minute(), 0, 0, loc)
					duration := time.Duration(endMin-startMin) * time.Minute
					interval := duration / time.Duration(count)

					for i := 0; i < count; i++ {
						select {
						case <-ctx.Done():
							return
						default:
						}

						t := windowStart.Add(interval * time.Duration(i))
						now := time.Now().In(loc)
						// Пропускаем выполнение, если расчётное время уже прошло
						if t.Before(now) {
							continue
						}
						if sleep := t.Sub(now); sleep > 0 {
							select {
							case <-time.After(sleep):
							case <-ctx.Done():
								return
							}
						}
						payload, _ := json.Marshal(act.RequestBody)
						req, err := http.NewRequestWithContext(ctx, "POST", act.URL, bytes.NewBuffer(payload))
						if err != nil {
							continue
						}
						req.Header.Set("Content-Type", "application/json")
						req.Header.Set("Authorization", "Bearer "+bearerToken)
						http.DefaultClient.Do(req)
					}
				}(act, cfg, day)
			case strings.Contains(act.URL, "reaction"):
				cfg := reactionCfg
				if len(cfg.DispatcherActivityMax) != 2 || len(cfg.DispatcherPeriod) != 2 {
					continue
				}

				wg.Add(1)
				go func(act ActivityRequest, cfg ActivitySettings, offset int) {
					defer wg.Done()

					// Парсим временные границы выполнения
					startTime, err1 := time.Parse("15:04", cfg.DispatcherPeriod[0])
					endTime, err2 := time.Parse("15:04", cfg.DispatcherPeriod[1])
					if err1 != nil || err2 != nil {
						return
					}
					startMin := startTime.Hour()*60 + startTime.Minute()
					endMin := endTime.Hour()*60 + endTime.Minute()
					minAct := cfg.DispatcherActivityMax[0]
					maxAct := cfg.DispatcherActivityMax[1]
					if endMin <= startMin || maxAct < minAct {
						return
					}

					count := rand.Intn(maxAct-minAct+1) + minAct

					currentDay := start.AddDate(0, 0, offset)
					// Начало окна активности в московском часовом поясе
					windowStart := time.Date(currentDay.Year(), currentDay.Month(), currentDay.Day(), startTime.Hour(), startTime.Minute(), 0, 0, loc)
					duration := time.Duration(endMin-startMin) * time.Minute
					interval := duration / time.Duration(count)

					for i := 0; i < count; i++ {
						select {
						case <-ctx.Done():
							return
						default:
						}

						t := windowStart.Add(interval * time.Duration(i))
						now := time.Now().In(loc)
						// Пропускаем выполнение, если расчётное время уже прошло
						if t.Before(now) {
							continue
						}
						if sleep := t.Sub(now); sleep > 0 {
							select {
							case <-time.After(sleep):
							case <-ctx.Done():
								return
							}
						}
						payload, _ := json.Marshal(act.RequestBody)
						req, err := http.NewRequestWithContext(ctx, "POST", act.URL, bytes.NewBuffer(payload))
						if err != nil {
							continue
						}
						req.Header.Set("Content-Type", "application/json")
						req.Header.Set("Authorization", "Bearer "+bearerToken)
						http.DefaultClient.Do(req)
					}
				}(act, cfg, day)
			case strings.Contains(act.URL, "unsubscribe"):
				if unsubscribeCfg.DispatcherStart == "" {
					continue
				}

				wg.Add(1)
				go func(act ActivityRequest, offset int) {
					defer wg.Done()
					runAtDispatcherStart(ctx, start, loc, act, unsubscribeCfg.DispatcherStart, offset)
				}(act, day)
			case strings.Contains(act.URL, "active_sessions_disconnect"):
				if disconnectCfg.DispatcherStart == "" {
					continue
				}

				wg.Add(1)
				go func(act ActivityRequest, offset int) {
					defer wg.Done()
					runAtDispatcherStart(ctx, start, loc, act, disconnectCfg.DispatcherStart, offset)
				}(act, day)
			default:
				continue
			}
		}

		wg.Wait()
	}
}

// runAtDispatcherStart выполняет запрос act в указанное время по МСК.
// Используем отдельную функцию, чтобы переиспользовать логику для разных типов действий.
func runAtDispatcherStart(ctx context.Context, base time.Time, loc *time.Location, act ActivityRequest, dispatcherStart string, offset int) {
	startTime, err := time.Parse("15:04", dispatcherStart)
	if err != nil {
		return
	}

	// Рассчитываем целевое время запуска на текущий день.
	currentDay := base.AddDate(0, 0, offset).In(loc)
	target := time.Date(currentDay.Year(), currentDay.Month(), currentDay.Day(), startTime.Hour(), startTime.Minute(), 0, 0, loc)

	now := time.Now().In(loc)
	if target.Before(now) {
		// Если время уже прошло, выполнять задачу бессмысленно.
		return
	}

	if sleep := target.Sub(now); sleep > 0 {
		select {
		case <-time.After(sleep):
		case <-ctx.Done():
			return
		}
	}

	payload, _ := json.Marshal(act.RequestBody)
	req, err := http.NewRequestWithContext(ctx, "POST", act.URL, bytes.NewBuffer(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	http.DefaultClient.Do(req)
}
