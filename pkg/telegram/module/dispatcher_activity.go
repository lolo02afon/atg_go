package module

import (
	"bytes"
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

// ModF_DispatcherActivity выполняет запросы активности в течение
// заданного количества суток.
func ModF_DispatcherActivity(daysNumber int, activities []ActivityRequest, commentCfg, reactionCfg ActivitySettings) {
	rand.Seed(time.Now().UnixNano())
	start := time.Now()

	for day := 0; day < daysNumber; day++ {
		var wg sync.WaitGroup

		for _, act := range activities {
			var cfg ActivitySettings
			switch {
			case strings.Contains(act.URL, "comment"):
				cfg = commentCfg
			case strings.Contains(act.URL, "reaction"):
				cfg = reactionCfg
			default:
				continue
			}

			if len(cfg.DispatcherActivityMax) != 2 || len(cfg.DispatcherPeriod) != 2 {
				continue
			}

			wg.Add(1)
			go func(act ActivityRequest, cfg ActivitySettings, offset int) {
				defer wg.Done()

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
				windowStart := time.Date(currentDay.Year(), currentDay.Month(), currentDay.Day(), startTime.Hour(), startTime.Minute(), 0, 0, time.Local)
				duration := time.Duration(endMin-startMin) * time.Minute
				interval := duration / time.Duration(count)

				for i := 0; i < count; i++ {
					t := windowStart.Add(interval * time.Duration(i))
					if sleep := time.Until(t); sleep > 0 {
						time.Sleep(sleep)
					}
					payload, _ := json.Marshal(act.RequestBody)
					req, err := http.NewRequest("POST", act.URL, bytes.NewBuffer(payload))
					if err != nil {
						continue
					}
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+bearerToken)
					http.DefaultClient.Do(req)
				}
			}(act, cfg, day)
		}

		wg.Wait()
	}
}
