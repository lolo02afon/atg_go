package storage

import (
	"atg_go/models"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type DB struct {
	Conn *sql.DB
}

func NewDB(conn *sql.DB) *DB {
	return &DB{Conn: conn}
}

func (db *DB) CreateProxy(p models.Proxy) (*models.Proxy, error) {
	query := `
               INSERT INTO proxy (ip, port, login, password, type, ipv6, is_active)
               VALUES ($1, $2, $3, $4, $5, $6, $7)
               RETURNING id, account_count
       `
	err := db.Conn.QueryRow(query, p.IP, p.Port, p.Login, p.Password, p.Type, p.IPv6, p.IsActive).Scan(&p.ID, &p.AccountsCount)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (db *DB) GetProxyByID(id int) (*models.Proxy, error) {
	var p models.Proxy
	var active sql.NullBool
	query := `
               SELECT id, ip, port, login, password, type, ipv6, account_count, is_active
               FROM proxy
               WHERE id = $1
       `
	err := db.Conn.QueryRow(query, id).Scan(
		&p.ID,
		&p.IP,
		&p.Port,
		&p.Login,
		&p.Password,
		&p.Type,
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

func (db *DB) CreateAccount(account models.Account) (*models.Account, error) {
	query := `
               INSERT INTO accounts (phone, api_id, api_hash, phone_code_hash, proxy_id)
               VALUES ($1, $2, $3, $4, $5)
               RETURNING id
       `

	err := db.Conn.QueryRow(
		query,
		account.Phone,
		account.ApiID,
		account.ApiHash,
		account.PhoneCodeHash,
		account.ProxyID,
	).Scan(&account.ID)

	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (db *DB) GetAccountByID(id int) (*models.Account, error) {
	var account models.Account
	account.Proxy = &models.Proxy{}
	var active sql.NullBool
	query := `
               SELECT a.id, a.phone, a.api_id, a.api_hash, a.phone_code_hash, a.is_authorized, a.proxy_id,
                      p.id, p.ip, p.port, p.login, p.password, p.type, p.ipv6, p.account_count, p.is_active
               FROM accounts a
               LEFT JOIN proxy p ON a.proxy_id = p.id
               WHERE a.id = $1
       `
	err := db.Conn.QueryRow(query, id).Scan(
		&account.ID,
		&account.Phone,
		&account.ApiID,
		&account.ApiHash,
		&account.PhoneCodeHash,
		&account.IsAuthorized,
		&account.ProxyID,
		&account.Proxy.ID,
		&account.Proxy.IP,
		&account.Proxy.Port,
		&account.Proxy.Login,
		&account.Proxy.Password,
		&account.Proxy.Type,
		&account.Proxy.IPv6,
		&account.Proxy.AccountsCount,
		&active,
	)
	if err != nil {
		return nil, err
	}
	account.Proxy.IsActive = active
	return &account, nil
}

func (db *DB) MarkAccountAsAuthorized(accountID int) error {
	_, err := db.Conn.Exec(
		"UPDATE accounts SET is_authorized = true WHERE id = $1",
		accountID,
	)
	return err
}

func (db *DB) GetAccountByPhone(phone string) (*models.Account, error) {
	var account models.Account
	var active sql.NullBool
	account.Proxy = &models.Proxy{}
	query := `
        SELECT a.id, a.phone, a.api_id, a.api_hash, a.phone_code_hash, a.is_authorized, a.proxy_id,
               p.id, p.ip, p.port, p.login, p.password, p.type, p.ipv6, p.account_count, p.is_active
        FROM accounts a
        LEFT JOIN proxy p ON a.proxy_id = p.id
        WHERE phone = $1
    `
	err := db.Conn.QueryRow(query, phone).Scan(
		&account.ID,
		&account.Phone,
		&account.ApiID,
		&account.ApiHash,
		&account.PhoneCodeHash,
		&account.IsAuthorized,
		&account.ProxyID,
		&account.Proxy.ID,
		&account.Proxy.IP,
		&account.Proxy.Port,
		&account.Proxy.Login,
		&account.Proxy.Password,
		&account.Proxy.Type,
		&account.Proxy.IPv6,
		&account.Proxy.AccountsCount,
		&active,
	)
	if err != nil {
		return nil, err
	}
	account.Proxy.IsActive = active
	return &account, nil
}

func (db *DB) AssignProxyToAccount(accountID, proxyID int, limit int) error {
	var count int
	if err := db.Conn.QueryRow("SELECT account_count FROM proxy WHERE id = $1", proxyID).Scan(&count); err != nil {
		return err
	}
	if limit > 0 && count >= limit {
		return fmt.Errorf("proxy limit reached")
	}
	_, err := db.Conn.Exec("UPDATE accounts SET proxy_id = $1 WHERE id = $2", proxyID, accountID)
	return err
}

// возвращает все авторизованные аккаунты
func (db *DB) GetAuthorizedAccounts() ([]models.Account, error) {
	// Запрос для выборки авторизованных аккаунтов
	query := `
        SELECT a.id, a.phone, a.api_id, a.api_hash, a.phone_code_hash, a.is_authorized, a.proxy_id,
               p.id, p.ip, p.port, p.login, p.password, p.type, p.ipv6, p.account_count, p.is_active
        FROM accounts a
        LEFT JOIN proxy p ON a.proxy_id = p.id
        WHERE a.is_authorized = true
    `

	// Выполняем запрос
	rows, err := db.Conn.Query(query)
	if err != nil {
		log.Printf("[DB ERROR] Failed to get authorized accounts: %v", err)
		return nil, fmt.Errorf("database error")
	}
	defer rows.Close()

	var accounts []models.Account

	// Итерируем по результатам
	for rows.Next() {
		var account models.Account
		var active sql.NullBool
		account.Proxy = &models.Proxy{}
		if err := rows.Scan(
			&account.ID,
			&account.Phone,
			&account.ApiID,
			&account.ApiHash,
			&account.PhoneCodeHash,
			&account.IsAuthorized,
			&account.ProxyID,
			&account.Proxy.ID,
			&account.Proxy.IP,
			&account.Proxy.Port,
			&account.Proxy.Login,
			&account.Proxy.Password,
			&account.Proxy.Type,
			&account.Proxy.IPv6,
			&account.Proxy.AccountsCount,
			&active,
		); err != nil {
			log.Printf("[DB WARN] Failed to scan account: %v", err)
			continue // Пропускаем проблемные записи
		}
		account.Proxy.IsActive = active
		accounts = append(accounts, account)
	}

	log.Printf("[DB INFO] Found %d authorized accounts", len(accounts))
	return accounts, nil
}
