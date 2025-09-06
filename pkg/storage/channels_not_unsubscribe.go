package storage

import "database/sql"

// GetChannelsNotUnsubscribeURLs возвращает список ссылок, от которых нельзя отписываться.
func (db *DB) GetChannelsNotUnsubscribeURLs() ([]string, error) {
	rows, err := db.Conn.Query(`SELECT url_channel FROM channels_not_unsubscribe`)
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
