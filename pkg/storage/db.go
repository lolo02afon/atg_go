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

func (db *DB) CreateAccount(account models.Account) (*models.Account, error) {
	query := `
		INSERT INTO accounts (phone, api_id, api_hash, phone_code_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	err := db.Conn.QueryRow(
		query,
		account.Phone,
		account.ApiID,
		account.ApiHash,
		account.PhoneCodeHash,
	).Scan(&account.ID)

	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (db *DB) GetAccountByID(id int) (*models.Account, error) {
	var account models.Account
	query := `
		SELECT id, phone, api_id, api_hash, phone_code_hash, is_authorized
		FROM accounts
		WHERE id = $1
	`
	err := db.Conn.QueryRow(query, id).Scan(
		&account.ID,
		&account.Phone,
		&account.ApiID,
		&account.ApiHash,
		&account.PhoneCodeHash,
		&account.IsAuthorized,
	)
	if err != nil {
		return nil, err
	}
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
	query := `
        SELECT id, phone, api_id, api_hash, phone_code_hash, is_authorized
        FROM accounts
        WHERE phone = $1
    `
	err := db.Conn.QueryRow(query, phone).Scan(
		&account.ID,
		&account.Phone,
		&account.ApiID,
		&account.ApiHash,
		&account.PhoneCodeHash,
		&account.IsAuthorized,
	)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// возвращает все авторизованные аккаунты
func (db *DB) GetAuthorizedAccounts() ([]models.Account, error) {
	// Запрос для выборки авторизованных аккаунтов
	query := `
        SELECT id, phone, api_id, api_hash, phone_code_hash, is_authorized
        FROM accounts
        WHERE is_authorized = true
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
		if err := rows.Scan(
			&account.ID,
			&account.Phone,
			&account.ApiID,
			&account.ApiHash,
			&account.PhoneCodeHash,
			&account.IsAuthorized,
		); err != nil {
			log.Printf("[DB WARN] Failed to scan account: %v", err)
			continue // Пропускаем проблемные записи
		}
		accounts = append(accounts, account)
	}

	log.Printf("[DB INFO] Found %d authorized accounts", len(accounts))
	return accounts, nil
}
