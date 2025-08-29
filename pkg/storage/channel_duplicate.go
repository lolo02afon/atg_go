package storage

import (
	"database/sql"

	"atg_go/models"
)

// ChannelDuplicateOrder объединяет запись дублирования с адресом нашего канала.
type ChannelDuplicateOrder struct {
	models.ChannelDuplicate
	OrderURL         string  // Ссылка на наш канал (Order.URLDefault)
	OrderChannelTGID *string // ID нашего канала
}

// GetChannelDuplicates возвращает список каналов-источников и связанные с ними заказы.
func (db *DB) GetChannelDuplicates() ([]ChannelDuplicateOrder, error) {
	rows, err := db.Conn.Query(`
                SELECT cd.id, cd.order_id, cd.url_channel_donor, cd.channel_donor_tgid, cd.post_text_remove, cd.post_text_add, cd.last_post_id,
                       o.url_default, o.channel_tgid
                FROM channel_duplicate cd
                JOIN orders o ON cd.order_id = o.id
                WHERE cd.url_channel_donor <> '' AND o.url_default <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ChannelDuplicateOrder
	for rows.Next() {
		var cd ChannelDuplicateOrder
		var (
			donorTGID, postRemove, postAdd, orderTGID sql.NullString
			lastPost                                  sql.NullInt64
		)
		if err := rows.Scan(&cd.ID, &cd.OrderID, &cd.URLChannelDonor, &donorTGID, &postRemove, &postAdd, &lastPost, &cd.OrderURL, &orderTGID); err != nil {
			return nil, err
		}
		if donorTGID.Valid {
			cd.ChannelDonorTGID = &donorTGID.String
		}
		if postRemove.Valid {
			cd.PostTextRemove = &postRemove.String
		}
		if postAdd.Valid {
			cd.PostTextAdd = &postAdd.String
		}
		if lastPost.Valid {
			v := int(lastPost.Int64)
			cd.LastPostID = &v
		}
		if orderTGID.Valid {
			cd.OrderChannelTGID = &orderTGID.String
		}
		list = append(list, cd)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// GetChannelDonorURLs возвращает список ссылок на каналы-доноры.
// Эти каналы используются при дублировании контента,
// поэтому отписка от них для мониторинговых аккаунтов запрещена.
func (db *DB) GetChannelDonorURLs() ([]string, error) {
	rows, err := db.Conn.Query(`SELECT url_channel_donor FROM channel_duplicate WHERE url_channel_donor <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var url sql.NullString
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		if url.Valid {
			urls = append(urls, url.String)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return urls, nil
}

// SetChannelDonorTGID сохраняет ID донорского канала.
func (db *DB) SetChannelDonorTGID(id int, tgid string) error {
	_, err := db.Conn.Exec(`UPDATE channel_duplicate SET channel_donor_tgid = $1 WHERE id = $2`, tgid, id)
	return err
}

// TrySetLastPostID обновляет last_post_id, если сообщение новое.
// Возвращает true, если обновление выполнено.
func (db *DB) TrySetLastPostID(id int, postID int) (bool, error) {
	res, err := db.Conn.Exec(`UPDATE channel_duplicate SET last_post_id = $2 WHERE id = $1 AND (last_post_id IS NULL OR last_post_id < $2)`, id, postID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
