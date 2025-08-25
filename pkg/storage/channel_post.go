package storage

import (
	"atg_go/models"
	"log"
)

// CreateChannelPost сохраняет информацию о новом посте канала в БД.
// Используется мониторингом для фиксации публикаций заказов.
func (db *DB) CreateChannelPost(p models.ChannelPost) error {
	_, err := db.Conn.Exec(`INSERT INTO channel_post (order_id, post_date_time, post_url) VALUES ($1, $2, $3)`,
		p.OrderID, p.PostDateTime, p.PostURL)
	if err != nil {
		log.Printf("[DB ERROR] сохранение поста: %v", err)
	}
	return err
}
