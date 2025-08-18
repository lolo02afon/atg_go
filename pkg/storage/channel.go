package storage

import "database/sql"

// GetChannelNames возвращает список названий всех каналов
// Используется для заполнения выпадающего списка категорий заказов
func (db *DB) GetChannelNames() ([]string, error) {
	rows, err := db.Conn.Query(`SELECT name FROM channels`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name sql.NullString
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if name.Valid {
			names = append(names, name.String)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return names, nil
}
