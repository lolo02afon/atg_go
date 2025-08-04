package module

import (
	"bytes"
	"net/http"
	"time"
)

// ModF_DispatcherActivity запускает отправку комментариев с заданным интервалом.
func ModF_DispatcherActivity(interval time.Duration, repeat int) {
	payload := []byte(`{"posts_count":5}`)
	for i := 0; i < repeat; i++ {
		http.Post("http://localhost:8080/comment/send", "application/json", bytes.NewBuffer(payload))
		if i < repeat-1 {
			time.Sleep(interval)
		}
	}
}
