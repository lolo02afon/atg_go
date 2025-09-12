package module

import (
	"context"
	"sync"

	"atg_go/pkg/storage"
)

// handler.go содержит базовый обработчик модуля Telegram.
// Здесь хранится общее состояние и доступ к БД, чтобы остальные обработчики
// могли запускать фоновые задачи и пользоваться одной точкой входа.
type Handler struct {
	DB *storage.DB

	mu    sync.Mutex
	tasks map[int]context.CancelFunc
	next  int
}

// NewHandler создаёт новый экземпляр обработчика.
func NewHandler(db *storage.DB) *Handler {
	return &Handler{DB: db, tasks: make(map[int]context.CancelFunc)}
}
