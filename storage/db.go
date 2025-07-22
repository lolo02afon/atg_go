package storage

import (
	"database/sql"

	"atg_go/models"

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
		INSERT INTO accounts (phone, api_id, api_hash)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	err := db.Conn.QueryRow(query, account.Phone, account.APIID, account.APIHash).Scan(&account.ID)
	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (db *DB) GetAccountByID(id int) (*models.Account, error) {
	var account models.Account
	query := `
		SELECT id, phone, api_id, api_hash
		FROM accounts
		WHERE id = $1
	`
	err := db.Conn.QueryRow(query, id).Scan(&account.ID, &account.Phone, &account.APIID, &account.APIHash)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (db *DB) CreateVerificationCode(code models.VerificationCode) (*models.VerificationCode, error) {
	query := `
		INSERT INTO verification_codes (account_id, code, is_verified)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err := db.Conn.QueryRow(query, code.AccountID, code.Code, code.IsVerified).Scan(&code.ID)
	if err != nil {
		return nil, err
	}
	return &code, nil
}

func (db *DB) UpdateVerificationCode(code models.VerificationCode) error {
	query := `
		UPDATE verification_codes
		SET code = $1, is_verified = $2
		WHERE id = $3
	`
	_, err := db.Conn.Exec(query, code.Code, code.IsVerified, code.ID)
	return err
}

func (db *DB) GetVerificationCodeByID(id int) (*models.VerificationCode, error) {
	query := `
		SELECT id, account_id, code, is_verified
		FROM verification_codes
		WHERE id = $1
	`

	var code models.VerificationCode
	err := db.Conn.QueryRow(query, id).Scan(&code.ID, &code.AccountID, &code.Code, &code.IsVerified)
	if err != nil {
		return nil, err
	}

	return &code, nil
}
