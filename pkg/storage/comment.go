package storage

import (
	"atg_go/models"
	"database/sql"
	"encoding/json"
	"log"
	"math/rand"
	"time"
)

type CommentDB struct {
	Conn *sql.DB
}

func NewCommentDB(conn *sql.DB) *CommentDB {
	return &CommentDB{Conn: conn}
}

// GetRandomChannel возвращает случайный URL канала
func (cdb *CommentDB) GetRandomChannel() (string, error) {
	var channel models.Channel

	// 1. Получаем общее количество записей
	var count int
	err := cdb.Conn.QueryRow("SELECT COUNT(*) FROM channels").Scan(&count)
	if err != nil {
		log.Printf("[DB ERROR] Channel count failed: %v", err)
		return "", err
	}

	if count == 0 {
		return "", sql.ErrNoRows
	}

	// 2. Выбираем случайную запись
	rand.Seed(time.Now().UnixNano())
	offset := rand.Intn(count)

	row := cdb.Conn.QueryRow(`
        SELECT id, name, urls 
        FROM channels 
        LIMIT 1 OFFSET $1
    `, offset)

	var urlsJSON []byte
	if err := row.Scan(&channel.ID, &channel.Name, &urlsJSON); err != nil {
		log.Printf("[DB ERROR] Channel scan failed: %v", err)
		return "", err
	}

	// 3. Парсим JSON
	if err := json.Unmarshal(urlsJSON, &channel.URLs); err != nil {
		log.Printf("[DB ERROR] URL parsing failed: %v", err)
		return "", err
	}

	if len(channel.URLs) == 0 {
		return "", sql.ErrNoRows
	}

	// 4. Выбираем случайный URL
	url := channel.URLs[rand.Intn(len(channel.URLs))]
	log.Printf("[DB] Selected channel: %s from group '%s'", url, channel.Name)

	return url, nil
}
