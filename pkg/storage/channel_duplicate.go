package storage

import (
	"database/sql"

	"atg_go/models"
	"github.com/lib/pq"
)

// rowScanner описывает минимальный набор методов для чтения строки.
// Это позволяет переиспользовать функцию сканирования как с *sql.Rows,
// так и с *sql.Row.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanChannelDuplicateOrder заполняет структуру ChannelDuplicateOrder из результата запроса.
// Общая логика вынесена в отдельную функцию, чтобы избежать дублирования кода.
func scanChannelDuplicateOrder(rs rowScanner) (ChannelDuplicateOrder, error) {
	var cd ChannelDuplicateOrder
	var (
		donorTGID, postRemove, postAdd, orderTGID sql.NullString
		postSkip                                  []byte // JSON с условиями пропуска постов
		lastPost                                  sql.NullInt64
		postCountDay                              pq.StringArray
	)
	if err := rs.Scan(&cd.ID, &cd.OrderID, &cd.URLChannelDonor, &donorTGID, &postRemove, &postAdd, &postSkip, &lastPost, &postCountDay, &cd.OrderURL, &orderTGID); err != nil {
		return cd, err
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
	cd.PostSkip = postSkip
	if lastPost.Valid {
		v := int(lastPost.Int64)
		cd.LastPostID = &v
	}
	cd.PostCountDay = postCountDay
	if orderTGID.Valid {
		cd.OrderChannelTGID = &orderTGID.String
	}
	return cd, nil
}

// ChannelDuplicateOrder объединяет запись дублирования с адресом нашего канала.
type ChannelDuplicateOrder struct {
	models.ChannelDuplicate
	OrderURL         string  // Ссылка на наш канал (Order.URLDefault)
	OrderChannelTGID *string // ID нашего канала
}

// GetChannelDuplicates возвращает список каналов-источников и связанные с ними заказы.
func (db *DB) GetChannelDuplicates() ([]ChannelDuplicateOrder, error) {
	rows, err := db.Conn.Query(`
                SELECT cd.id, cd.order_id, cd.url_channel_donor, cd.channel_donor_tgid, cd.post_text_remove, cd.post_text_add, cd.post_skip, cd.last_post_id, cd.post_count_day,
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
		cd, err := scanChannelDuplicateOrder(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, cd)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// GetChannelDuplicateOrderByID возвращает запись дублирования с привязанным заказом по её ID.
func (db *DB) GetChannelDuplicateOrderByID(id int) (*ChannelDuplicateOrder, error) {
	row := db.Conn.QueryRow(`
                SELECT cd.id, cd.order_id, cd.url_channel_donor, cd.channel_donor_tgid, cd.post_text_remove, cd.post_text_add, cd.post_skip, cd.last_post_id, cd.post_count_day,
                       o.url_default, o.channel_tgid
                FROM channel_duplicate cd
                JOIN orders o ON cd.order_id = o.id
                WHERE cd.id = $1`, id)
	cd, err := scanChannelDuplicateOrder(row)
	if err != nil {
		return nil, err
	}
	return &cd, nil
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

// TrySetLastPostID обновляет last_post_id и возвращает актуальные тексты для обработки.
// Если запись не изменилась, updated будет false, а строки равны nil.
func (db *DB) TrySetLastPostID(id int, postID int) (updated bool, remove *string, add *string, err error) {
	row := db.Conn.QueryRow(`UPDATE channel_duplicate SET last_post_id = $2 WHERE id = $1 AND (last_post_id IS NULL OR last_post_id < $2) RETURNING post_text_remove, post_text_add`, id, postID)
	var (
		postRemove sql.NullString
		postAdd    sql.NullString
	)
	if err = row.Scan(&postRemove, &postAdd); err != nil {
		if err == sql.ErrNoRows {
			// Обновление не требуется
			return false, nil, nil, nil
		}
		return false, nil, nil, err
	}
	if postRemove.Valid {
		remove = &postRemove.String
	}
	if postAdd.Valid {
		add = &postAdd.String
	}
	return true, remove, add, nil
}

// UpdateChannelDuplicateTimes обновляет расписание публикаций (post_count_day) для записи channel_duplicate.
func (db *DB) UpdateChannelDuplicateTimes(id int, times pq.StringArray) error {
	_, err := db.Conn.Exec(
		`UPDATE channel_duplicate SET post_count_day = $1 WHERE id = $2`,
		pq.Array(times), id,
	)
	return err
}

// GetPostReactionsForOrder возвращает список реакций для постов указанного заказа.
// При отсутствии записи или NULL возвращает nil без ошибки.
func (db *DB) GetPostReactionsForOrder(orderID int) (pq.StringArray, error) {
	var raw sql.NullString
	// Приводим массив к тексту, чтобы получить строку вида {"😀","😢"}
	err := db.Conn.QueryRow(
		`SELECT post_reactions::text FROM orders WHERE id = $1`,
		orderID,
	).Scan(&raw)
	if err == sql.ErrNoRows || !raw.Valid {
		// Поле отсутствует или равно NULL — используем стандартную логику
		return nil, nil
	}
	var reactions pq.StringArray
	// pq.StringArray умеет разбирать строку с фигурными скобками
	if err := reactions.Scan(raw.String); err != nil {
		return nil, err
	}
	return reactions, nil
}
