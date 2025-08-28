package storage

import "database/sql"

// GetCategoryNames возвращает список названий всех категорий
// Используется для заполнения выпадающего списка категорий заказов
func (db *DB) GetCategoryNames() ([]string, error) {
	rows, err := db.Conn.Query(`SELECT name FROM categories`)
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
