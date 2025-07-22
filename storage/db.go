package storage

import (
	"database/sql"
	"log"

	"atg_go/models"

	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

// Подключение к БД
func NewDB(dataSource string) *DB {
	db, err := sql.Open("postgres", dataSource)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatalf("Ошибка при попытке соединения: %v", err)
	}
	return &DB{conn: db}
}

// Создание аккаунта
func (db *DB) CreateAccount(account models.Account) (models.Account, error) {
	query := `
		INSERT INTO accounts (phone, api_id, api_hash, is_authorized)
		VALUES ($1, $2, $3, $4)
		RETURNING id;
	`
	err := db.conn.QueryRow(query, account.Phone, account.ApiID, account.ApiHash, account.IsAuthorized).Scan(&account.ID)
	return account, err
}

// Сохранение кода подтверждения
func (db *DB) CreateVerificationCode(code models.VerificationCode) (models.VerificationCode, error) {
	query := `INSERT INTO verification_codes (account_id, code, is_verified, send)
	          VALUES ($1, $2, $3, $4) RETURNING id;`
	err := db.conn.QueryRow(query, code.AccountID, code.Code, code.IsVerified, code.Send).Scan(&code.ID)
	return code, err
}

// Получение аккаунта по id
func (db *DB) UpdateVerificationCode(code models.VerificationCode) error {
	query := `UPDATE verification_codes SET code = $1, is_verified = $2, send = $3 WHERE id = $4`
	_, err := db.conn.Exec(query, code.Code, code.IsVerified, code.Send, code.ID)
	return err
}
