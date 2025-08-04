package module

import (
	"bytes"
	"net/http"
	"time"
)

// ModF_DispatcherActivity запускает отправку комментариев каждые 30 секунд
// на протяжении двух циклов.
func ModF_DispatcherActivity() {
	payload := []byte(`{"posts_count":5}`)
	for i := 0; i < 2; i++ {
		http.Post("http://localhost:8080/comment/send", "application/json", bytes.NewBuffer(payload))
		if i < 1 {
			time.Sleep(30 * time.Second)
		}
	}
}
