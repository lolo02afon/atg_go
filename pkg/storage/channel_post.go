package storage

import (
	"atg_go/models"
	"log"
)

// CreateChannelPost сохраняет информацию о новом посте канала в БД и возвращает идентификатор записи.
// Используется мониторингом для фиксации публикаций заказов с расчётом активной аудитории.
func (db *DB) CreateChannelPost(p models.ChannelPost) (int, error) {
	var id int
	err := db.Conn.QueryRow(`INSERT INTO channel_post (order_id, post_date_time, post_url, subs_active_view, subs_active_reaction, subs_active_repost) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		p.OrderID, p.PostDateTime, p.PostURL, p.SubsActiveView, p.SubsActiveReaction, p.SubsActiveRepost).Scan(&id)
	if err != nil {
		log.Printf("[DB ERROR] сохранение поста: %v", err)
		return 0, err
	}
	return id, nil
}
