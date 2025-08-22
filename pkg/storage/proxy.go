package storage

import (
	"atg_go/models"
	"database/sql"
)

// CreateProxy сохраняет прокси, чтобы его можно было переиспользовать без дублирования
// данных и лишних обращений к внешним сервисам.
func (db *DB) CreateProxy(p models.Proxy) (*models.Proxy, error) {
	query := `
              INSERT INTO proxy (ip, port, login, password, ipv6, is_active)
              VALUES ($1, $2, $3, $4, $5, $6)
              RETURNING id, account_count
       `
	err := db.Conn.QueryRow(query, p.IP, p.Port, p.Login, p.Password, p.IPv6, p.IsActive).Scan(&p.ID, &p.AccountsCount)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetProxyByID загружает прокси по идентификатору, чтобы быстро получать его
// параметры без повторного ввода пользователем.
func (db *DB) GetProxyByID(id int) (*models.Proxy, error) {
	var p models.Proxy
	var active sql.NullBool
	query := `
              SELECT id, ip, port, login, password, ipv6, account_count, is_active
              FROM proxy
              WHERE id = $1
       `
	err := db.Conn.QueryRow(query, id).Scan(
		&p.ID,
		&p.IP,
		&p.Port,
		&p.Login,
		&p.Password,
		&p.IPv6,
		&p.AccountsCount,
		&active,
	)
	if err != nil {
		return nil, err
	}
	p.IsActive = active
	return &p, nil
}
