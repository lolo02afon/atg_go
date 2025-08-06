package module

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// ActivityRequest описывает один запрос активности с адресом и произвольным телом.
type ActivityRequest struct {
	URL         string         `json:"url"`
	RequestBody map[string]any `json:"request_body"`
}

// ModF_DispatcherActivity выполняет указанные запросы с заданным интервалом и количеством повторений.
func ModF_DispatcherActivity(interval time.Duration, repeat int, activities []ActivityRequest) {
	for i := 0; i < repeat; i++ {
		for _, act := range activities {
			payload, _ := json.Marshal(act.RequestBody)
			http.Post(act.URL, "application/json", bytes.NewBuffer(payload))
		}
		if i < repeat-1 {
			time.Sleep(interval)
		}
	}
}
