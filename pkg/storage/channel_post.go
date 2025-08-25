package storage

import (
	"atg_go/models"
	"log"
)

// CreateChannelPost сохраняет информацию о новом посте канала в БД.
// Используется мониторингом для фиксации публикаций заказов с расчётом активной аудитории.
func (db *DB) CreateChannelPost(p models.ChannelPost) error {
	_, err := db.Conn.Exec(`INSERT INTO channel_post (order_id, post_date_time, post_url, subs_active_view, subs_active_reaction, subs_active_repost) VALUES ($1, $2, $3, $4, $5, $6)`,
		p.OrderID, p.PostDateTime, p.PostURL, p.SubsActiveView, p.SubsActiveReaction, p.SubsActiveRepost)
	if err != nil {
		log.Printf("[DB ERROR] сохранение поста: %v", err)
	}
	return err
}
