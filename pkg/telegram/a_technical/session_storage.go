package module

import (
	"context"
	"database/sql"
	"log"

	"github.com/gotd/td/session"
)

// DBSessionStorage хранит и загружает сессии Telegram из таблицы account_session.
type DBSessionStorage struct {
	DB        *sql.DB
	AccountID int
}

// LoadSession загружает текст сессии из БД.
func (s *DBSessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	if s == nil || s.DB == nil {
		return nil, session.ErrNotFound
	}

	var data string
	// В таблице account_session хранится не более одной записи на аккаунт,
	// поэтому достаточно выбрать её без сортировки.
	err := s.DB.QueryRowContext(ctx, "SELECT data_json FROM account_session WHERE account = $1", s.AccountID).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, session.ErrNotFound
	}
	if err != nil {
		log.Printf("[DBSessionStorage] ошибка чтения сессии: %v", err)
		return nil, err
	}
	return []byte(data), nil
}

// StoreSession сохраняет текст сессии в БД.
func (s *DBSessionStorage) StoreSession(ctx context.Context, data []byte) error {
	if s == nil || s.DB == nil {
		return session.ErrNotFound
	}
	// Обновляем существующую запись сессии, чтобы не создавать дубликаты
	_, err := s.DB.ExecContext(
		ctx,
		"INSERT INTO account_session (account, data_json) VALUES ($1, $2) "+
			"ON CONFLICT (account) DO UPDATE SET data_json = EXCLUDED.data_json, date_time = NOW()",
		s.AccountID,
		string(data),
	)
	if err != nil {
		log.Printf("[DBSessionStorage] ошибка сохранения сессии: %v", err)
		return err
	}
	return nil
}
