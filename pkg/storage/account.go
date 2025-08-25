package storage

import (
	"atg_go/models"
	"database/sql"
	"fmt"
	"log"

	"github.com/lib/pq"
)

// CreateAccount записывает аккаунт в БД, чтобы в дальнейшем
// не приходилось заново вводить параметры.
func (db *DB) CreateAccount(account models.Account) (*models.Account, error) {
	query := `
              INSERT INTO accounts (phone, api_id, api_hash, phone_code_hash, proxy_id, gender)
              VALUES ($1, $2, $3, $4, $5, $6)
              RETURNING id
       `

	// Фильтруем список полов через общую функцию, чтобы избежать дублирования логики
	gender := models.FilterGenders(account.Gender)

	err := db.Conn.QueryRow(
		query,
		account.Phone,
		account.ApiID,
		account.ApiHash,
		account.PhoneCodeHash,
		account.ProxyID,
		pq.Array(gender),
	).Scan(&account.ID)

	if err != nil {
		log.Printf("[DB ERROR] Ошибка при создании аккаунта: %v", err)
		return nil, err
	}

	log.Printf("[DB INFO] Аккаунт создан с ID=%d", account.ID)
	return &account, nil
}

// GetAccountByID возвращает аккаунт вместе с привязкой к прокси,
// чтобы сервисы могли работать с полными данными.
func (db *DB) GetAccountByID(id int) (*models.Account, error) {
	var account models.Account

	var (
		proxyID       sql.NullInt64
		proxyIP       sql.NullString
		proxyPort     sql.NullInt64
		proxyLogin    sql.NullString
		proxyPassword sql.NullString
		proxyIPv6     sql.NullString
		proxyCount    sql.NullInt64
		proxyActive   sql.NullBool
	)

	// Приводим gender к text[], чтобы можно было сканировать напрямую без вспомогательных типов
	query := `
              SELECT a.id, a.phone, a.api_id, a.api_hash, a.phone_code_hash, a.is_authorized, a.gender::text[], a.proxy_id, a.order_id,
                     p.id, p.ip, p.port, p.login, p.password, p.ipv6, p.account_count, p.is_active
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
		&account.Gender, // драйвер сам преобразует массив text[] в срез строк
		&account.ProxyID,
		&account.OrderID,
		&proxyID,
		&proxyIP,
		&proxyPort,
		&proxyLogin,
		&proxyPassword,
		&proxyIPv6,
		&proxyCount,
		&proxyActive,
	)
	if err != nil {
		return nil, err
	}

	if proxyID.Valid {
		account.Proxy = &models.Proxy{ID: int(proxyID.Int64), IsActive: proxyActive}
		if proxyIP.Valid {
			account.Proxy.IP = proxyIP.String
		}
		if proxyPort.Valid {
			account.Proxy.Port = int(proxyPort.Int64)
		}
		if proxyLogin.Valid {
			account.Proxy.Login = proxyLogin.String
		}
		if proxyPassword.Valid {
			account.Proxy.Password = proxyPassword.String
		}
		if proxyIPv6.Valid {
			account.Proxy.IPv6 = proxyIPv6.String
		}
		if proxyCount.Valid {
			account.Proxy.AccountsCount = int(proxyCount.Int64)
		}
	} else {
		account.Proxy = nil
		account.ProxyID = nil
	}

	return &account, nil
}

// GetLastAccount нужен, чтобы использовать свежесозданный аккаунт
// без дополнительного запроса его идентификатора.
func (db *DB) GetLastAccount() (*models.Account, error) {
	var account models.Account

	// Переменные для возможных NULL-значений связанных с прокси
	var (
		proxyID       sql.NullInt64
		proxyIP       sql.NullString
		proxyPort     sql.NullInt64
		proxyLogin    sql.NullString
		proxyPassword sql.NullString
		proxyIPv6     sql.NullString
		proxyCount    sql.NullInt64
		proxyActive   sql.NullBool
	)

	// Каст к text[] избавляет от необходимости использовать pq.Array при чтении
	query := `
              SELECT a.id, a.phone, a.api_id, a.api_hash, a.phone_code_hash, a.is_authorized, a.gender::text[], a.proxy_id, a.order_id,
                     p.id, p.ip, p.port, p.login, p.password, p.ipv6, p.account_count, p.is_active
              FROM accounts a
              LEFT JOIN proxy p ON a.proxy_id = p.id
              ORDER BY a.id DESC
              LIMIT 1
       `

	// Получаем последнюю запись из таблицы accounts
	err := db.Conn.QueryRow(query).Scan(
		&account.ID,
		&account.Phone,
		&account.ApiID,
		&account.ApiHash,
		&account.PhoneCodeHash,
		&account.IsAuthorized,
		&account.Gender, // массив читается напрямую
		&account.ProxyID,
		&account.OrderID,
		&proxyID,
		&proxyIP,
		&proxyPort,
		&proxyLogin,
		&proxyPassword,
		&proxyIPv6,
		&proxyCount,
		&proxyActive,
	)
	if err != nil {
		return nil, err
	}

	if proxyID.Valid {
		account.Proxy = &models.Proxy{ID: int(proxyID.Int64), IsActive: proxyActive}
		if proxyIP.Valid {
			account.Proxy.IP = proxyIP.String
		}
		if proxyPort.Valid {
			account.Proxy.Port = int(proxyPort.Int64)
		}
		if proxyLogin.Valid {
			account.Proxy.Login = proxyLogin.String
		}
		if proxyPassword.Valid {
			account.Proxy.Password = proxyPassword.String
		}
		if proxyIPv6.Valid {
			account.Proxy.IPv6 = proxyIPv6.String
		}
		if proxyCount.Valid {
			account.Proxy.AccountsCount = int(proxyCount.Int64)
		}
	} else {
		account.Proxy = nil
		account.ProxyID = nil
	}

	return &account, nil
}

// MarkAccountAsAuthorized фиксирует факт авторизации, чтобы другие сервисы
// понимали, что сессия активна.
func (db *DB) MarkAccountAsAuthorized(accountID int) error {
	_, err := db.Conn.Exec(
		"UPDATE accounts SET is_authorized = true WHERE id = $1",
		accountID,
	)
	return err
}

// MarkAccountAsUnauthorized сбрасывает флаг авторизации,
// чтобы в БД отражалось фактическое отсутствие рабочей сессии.
func (db *DB) MarkAccountAsUnauthorized(accountID int) error {
	_, err := db.Conn.Exec(
		"UPDATE accounts SET is_authorized = false WHERE id = $1",
		accountID,
	)
	return err
}

// UpdatePhoneCodeHash обновляет hash, чтобы повторно не запрашивать код у пользователя.
func (db *DB) UpdatePhoneCodeHash(accountID int, hash string) error {
	_, err := db.Conn.Exec(
		"UPDATE accounts SET phone_code_hash = $1 WHERE id = $2",
		hash,
		accountID,
	)
	return err
}

// GetAccountByPhone ищет аккаунт по номеру, чтобы избежать создания дубликатов.
func (db *DB) GetAccountByPhone(phone string) (*models.Account, error) {
	var account models.Account

	var (
		proxyID       sql.NullInt64
		proxyIP       sql.NullString
		proxyPort     sql.NullInt64
		proxyLogin    sql.NullString
		proxyPassword sql.NullString
		proxyIPv6     sql.NullString
		proxyCount    sql.NullInt64
		proxyActive   sql.NullBool
	)

	// Кастомное приведение gender к text[] упрощает чтение из массива
	query := `
       SELECT a.id, a.phone, a.api_id, a.api_hash, a.phone_code_hash, a.is_authorized, a.gender::text[], a.proxy_id, a.order_id,
              p.id, p.ip, p.port, p.login, p.password, p.ipv6, p.account_count, p.is_active
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
		&account.Gender, // используем прямое сканирование массива
		&account.ProxyID,
		&account.OrderID,
		&proxyID,
		&proxyIP,
		&proxyPort,
		&proxyLogin,
		&proxyPassword,
		&proxyIPv6,
		&proxyCount,
		&proxyActive,
	)
	if err != nil {
		return nil, err
	}

	if proxyID.Valid {
		account.Proxy = &models.Proxy{ID: int(proxyID.Int64), IsActive: proxyActive}
		if proxyIP.Valid {
			account.Proxy.IP = proxyIP.String
		}
		if proxyPort.Valid {
			account.Proxy.Port = int(proxyPort.Int64)
		}
		if proxyLogin.Valid {
			account.Proxy.Login = proxyLogin.String
		}
		if proxyPassword.Valid {
			account.Proxy.Password = proxyPassword.String
		}
		if proxyIPv6.Valid {
			account.Proxy.IPv6 = proxyIPv6.String
		}
		if proxyCount.Valid {
			account.Proxy.AccountsCount = int(proxyCount.Int64)
		}
	} else {
		account.Proxy = nil
		account.ProxyID = nil
	}

	return &account, nil
}

// AssignProxyToAccount привязывает прокси к аккаунту, учитывая лимит
// количества аккаунтов на одном прокси, чтобы не перегружать его.
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

// getAccounts возвращает список аккаунтов по произвольному условию WHERE.
// Это позволяет переиспользовать код выборки для разных типов аккаунтов
// и не дублировать логику обработки NULL-полей.
func (db *DB) getAccounts(where string, args ...any) ([]models.Account, error) {
	query := `
      SELECT a.id, a.phone, a.api_id, a.api_hash, a.phone_code_hash, a.is_authorized, a.gender::text[], a.proxy_id, a.order_id,
             p.id, p.ip, p.port, p.login, p.password, p.ipv6, p.account_count, p.is_active
      FROM accounts a
      LEFT JOIN proxy p ON a.proxy_id = p.id
      WHERE ` + where

	rows, err := db.Conn.Query(query, args...)
	if err != nil {
		log.Printf("[DB ERROR] Failed to get accounts: %v", err)
		return nil, fmt.Errorf("database error")
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var account models.Account
		var (
			proxyID        sql.NullInt64
			proxyIP        sql.NullString
			proxyPort      sql.NullInt64
			proxyLogin     sql.NullString
			proxyPassword  sql.NullString
			proxyIPv6      sql.NullString
			proxyCount     sql.NullInt64
			proxyIsActive  sql.NullBool
			accountProxyID sql.NullInt64
			accountOrderID sql.NullInt64
		)

		if err := rows.Scan(
			&account.ID,
			&account.Phone,
			&account.ApiID,
			&account.ApiHash,
			&account.PhoneCodeHash,
			&account.IsAuthorized,
			&account.Gender,
			&accountProxyID,
			&accountOrderID,
			&proxyID,
			&proxyIP,
			&proxyPort,
			&proxyLogin,
			&proxyPassword,
			&proxyIPv6,
			&proxyCount,
			&proxyIsActive,
		); err != nil {
			log.Printf("[DB WARN] Failed to scan account: %v", err)
			continue
		}

		if accountProxyID.Valid {
			id := int(accountProxyID.Int64)
			account.ProxyID = &id
		}
		if accountOrderID.Valid {
			id := int(accountOrderID.Int64)
			account.OrderID = &id
		}

		if proxyID.Valid {
			account.Proxy = &models.Proxy{
				ID:            int(proxyID.Int64),
				IP:            proxyIP.String,
				Port:          int(proxyPort.Int64),
				Login:         proxyLogin.String,
				Password:      proxyPassword.String,
				IPv6:          proxyIPv6.String,
				AccountsCount: int(proxyCount.Int64),
				IsActive:      proxyIsActive,
			}
		}

		accounts = append(accounts, account)
	}
	return accounts, nil
}

// GetAuthorizedAccounts возвращает все авторизованные аккаунты без мониторинга,
// чтобы сервисы могли быстро получить список рабочих сессий.
func (db *DB) GetAuthorizedAccounts() ([]models.Account, error) {
	accounts, err := db.getAccounts("a.is_authorized = true AND a.account_monitoring = false")
	if err == nil {
		log.Printf("[DB INFO] Found %d authorized accounts", len(accounts))
	}
	return accounts, err
}

// GetMonitoringAccounts возвращает авторизованные аккаунты,
// помеченные как мониторинговые.
func (db *DB) GetMonitoringAccounts() ([]models.Account, error) {
	return db.getAccounts("a.is_authorized = true AND a.account_monitoring = true")
}

// ReleaseMonitoringAccounts снимает привязку заказов с аккаунтов под мониторингом.
// Это нужно, чтобы такие аккаунты не выполняли заказы даже если были назначены ранее.
func (db *DB) ReleaseMonitoringAccounts() error {
	if _, err := db.Conn.Exec(`UPDATE accounts SET order_id = NULL WHERE account_monitoring = TRUE AND order_id IS NOT NULL`); err != nil {
		log.Printf("[DB ERROR] освобождение мониторинговых аккаунтов: %v", err)
		return err
	}
	return nil
}
