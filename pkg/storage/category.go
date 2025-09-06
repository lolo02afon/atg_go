package storage

import (
	"database/sql"
	"encoding/json"
	"log"

	"atg_go/models"

	"github.com/lib/pq"
)

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

// CreateCategory добавляет новую категорию с набором ссылок на каналы.
// Ссылки сохраняются в JSONB, поэтому предварительно кодируем их в JSON.
func (db *DB) CreateCategory(name string, urls []string) (*models.Category, error) {
	// Сохраняем ссылки в отдельной таблице, исключая дубли.
	if len(urls) > 0 {
		// Вставляем только уникальные ссылки; существующие записи игнорируются.
		if _, err := db.Conn.Exec(`
                       INSERT INTO channels (url)
                       SELECT DISTINCT unnest($1::text[])
                       ON CONFLICT (url) DO NOTHING
               `, pq.Array(urls)); err != nil {
			// Фиксируем проблему сохранения списка каналов
			log.Printf("[DB ERROR] вставка каналов %v завершилась ошибкой: %v", urls, err)
			return nil, err
		}
	}

	// Готовим JSON для хранения ссылок в категории.
	data, err := json.Marshal(urls)
	if err != nil {
		return nil, err
	}

	// Сохраняем категорию и возвращаем её идентификатор.
	var id int
	err = db.Conn.QueryRow(`INSERT INTO categories (name, urls) VALUES ($1, $2) RETURNING id`, name, data).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &models.Category{ID: id, Name: name, URLs: urls}, nil
}
