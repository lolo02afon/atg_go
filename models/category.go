package models

// Category представляет группу каналов в БД
type Category struct {
	ID   int      `json:"id"`
	Name string   `json:"name"`
	URLs []string `json:"urls"` // Массив URL вместо JSONB для удобства
}
