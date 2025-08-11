package module

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// ActivityRequest описывает один запрос активности с адресом и произвольным телом.
type ActivityRequest struct {
	URL         string         `json:"url"`
	RequestBody map[string]any `json:"request_body"`
}

// ModF_DispatcherActivity выполняет запросы активности, распределяя их по суткам и указанным временным окнам.
// daysNumber задаёт, сколько суток подряд выполняется активность.
func ModF_DispatcherActivity(daysNumber int, activities []ActivityRequest) {
	rand.Seed(time.Now().UnixNano())

	start := time.Now()

	for day := 0; day < daysNumber; day++ {
		var wg sync.WaitGroup

		for _, act := range activities {
			wg.Add(1)

			go func(act ActivityRequest, offset int) {
				defer wg.Done()

				period, ok1 := act.RequestBody["dispatcher_period"].([]interface{})
				maxRange, ok2 := act.RequestBody["dispatcher_activity_max"].([]interface{})
				if !ok1 || !ok2 || len(period) != 2 || len(maxRange) != 2 {
					return
				}

				// извлекаем часы начала и конца окна
				startHour := int(period[0].(float64))
				endHour := int(period[1].(float64))
				minAct := int(maxRange[0].(float64))
				maxAct := int(maxRange[1].(float64))

				if endHour <= startHour || maxAct < minAct {
					return
				}

				count := rand.Intn(maxAct-minAct+1) + minAct

				currentDay := start.AddDate(0, 0, offset)
				windowStart := time.Date(currentDay.Year(), currentDay.Month(), currentDay.Day(), startHour, 0, 0, 0, time.Local)
				duration := time.Duration(endHour-startHour) * time.Hour
				interval := duration / time.Duration(count)

				for i := 0; i < count; i++ {
					t := windowStart.Add(interval * time.Duration(i))
					if sleep := time.Until(t); sleep > 0 {
						time.Sleep(sleep)
					}
					payload, _ := json.Marshal(act.RequestBody)
					http.Post(act.URL, "application/json", bytes.NewBuffer(payload))
				}
			}(act, day)
		}

		wg.Wait()
	}
}
