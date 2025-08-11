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

				startStr, okStart := period[0].(string)
				endStr, okEnd := period[1].(string)
				if !okStart || !okEnd {
					return
				}

				startTime, err1 := time.Parse("15:04", startStr)
				endTime, err2 := time.Parse("15:04", endStr)
				if err1 != nil || err2 != nil {
					return
				}
				startMin := startTime.Hour()*60 + startTime.Minute()
				endMin := endTime.Hour()*60 + endTime.Minute()
				minAct := int(maxRange[0].(float64))
				maxAct := int(maxRange[1].(float64))

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
					http.Post(act.URL, "application/json", bytes.NewBuffer(payload))
				}
			}(act, day)
		}

		wg.Wait()
	}
}
