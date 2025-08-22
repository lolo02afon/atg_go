package storage

import (
	"atg_go/models"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/lib/pq"
)

type CommentDB struct {
	Conn *sql.DB
}

func NewCommentDB(conn *sql.DB) *CommentDB {
	return &CommentDB{Conn: conn}
}

// ErrNoChannel сообщает, что в базе нет ни одного канала,
// подходящего под категории заказа.
// Такое выделение ошибки позволяет обработчикам выдавать 404 и не шуметь в логах.
var ErrNoChannel = errors.New("no channel for order")

// GetRandomChannel выбирает случайный URL из каналов, подходящих под категории заказа.
// Мы фильтруем по заказу, чтобы аккаунт не отправлял активность в чужие тематики.
func (cdb *CommentDB) GetRandomChannel(orderID int) (string, error) {
	// 1. Загружаем список категорий заказа, иначе не сможем сузить выбор каналов.
	var categories pq.StringArray
	if err := cdb.Conn.QueryRow(`SELECT category FROM orders WHERE id = $1`, orderID).Scan(&categories); err != nil {
		log.Printf("[DB ERROR] получение категорий заказа %d: %v", orderID, err)
		return "", err
	}
	if len(categories) == 0 {
		// Заказ без категорий не даёт нам критериев отбора, поэтому дальше идти бессмысленно.
		return "", sql.ErrNoRows
	}

	// 2. Случайно выбираем один канал из подходящих по категориям.
	row := cdb.Conn.QueryRow(`
        SELECT id, name, urls
        FROM channels
        WHERE name = ANY($1)
        ORDER BY RANDOM()
        LIMIT 1
    `, pq.Array(categories))

	var channel models.Channel
	var urlsJSON []byte
	if err := row.Scan(&channel.ID, &channel.Name, &urlsJSON); err != nil {
		// Передаём ошибку дальше, чтобы вызывающая сторона могла решить, как реагировать на пустую выборку.
		log.Printf("[DB ERROR] выбор канала по заказу %d: %v", orderID, err)
		return "", err
	}

	// 3. Превращаем JSON-массив ссылок в удобный для случайного выбора срез.
	if err := json.Unmarshal(urlsJSON, &channel.URLs); err != nil {
		log.Printf("[DB ERROR] парсинг ссылок канала %d: %v", channel.ID, err)
		return "", err
	}
	if len(channel.URLs) == 0 {
		// Без ссылок канал не пригоден для активности.
		return "", sql.ErrNoRows
	}

	// 4. Выбираем одну ссылку случайным образом, чтобы равномерно распределять нагрузку по URL.
	rand.Seed(time.Now().UnixNano())
	url := channel.URLs[rand.Intn(len(channel.URLs))]
	log.Printf("[DB] выбран канал %s для заказа %d", url, orderID)

	return url, nil
}

// PickRandomChannel выбирает канал для отправки, если это возможно.
// Возвращаемая ошибка приводит sql.ErrNoRows к ErrNoChannel, чтобы вызовы вне
// пакета знали, когда каналов нет, и могли корректно ответить 404.
func PickRandomChannel(commentDB *CommentDB, orderID int) (string, error) {
	url, err := commentDB.GetRandomChannel(orderID)
	if errors.Is(err, sql.ErrNoRows) {
		// Преобразуем стандартную ошибку отсутствия строки в доменную
		// ErrNoChannel, чтобы не раскрывать детали хранилища наружу.
		return "", ErrNoChannel
	}
	if err != nil {
		return "", err
	}
	return url, nil
}
