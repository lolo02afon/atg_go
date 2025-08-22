package storage

import "database/sql"

// DB инкапсулирует соединение с БД, чтобы упростить передачу его
// по слоям приложения.
type DB struct {
	Conn *sql.DB
}

// NewDB создаёт обёртку над соединением для единообразной работы с БД.
func NewDB(conn *sql.DB) *DB {
	return &DB{Conn: conn}
}
