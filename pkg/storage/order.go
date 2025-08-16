package storage

import (
	"atg_go/models"
	"database/sql"
	"log"
)

// CreateOrder создаёт заказ и распределяет свободные аккаунты
func (db *DB) CreateOrder(o models.Order) (*models.Order, error) {
	tx, err := db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Вставляем запись о заказе
	err = tx.QueryRow(
		`INSERT INTO orders (name, url, accounts_number_theory) VALUES ($1, $2, $3) RETURNING id, accounts_number_fact, date_time`,
		o.Name, o.URL, o.AccountsNumberTheory,
	).Scan(&o.ID, &o.AccountsNumberFact, &o.DateTime)
	if err != nil {
		return nil, err
	}

	// Выбираем свободные аккаунты в случайном порядке
	rows, err := tx.Query(
		`SELECT id FROM accounts WHERE order_id IS NULL ORDER BY RANDOM() LIMIT $1`,
		o.AccountsNumberTheory,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`UPDATE accounts SET order_id = $1 WHERE id = $2`, o.ID, id); err != nil {
			return nil, err
		}
		count++
	}

	o.AccountsNumberFact = count

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	log.Printf("[DB INFO] Создан заказ %d, аккаунтов назначено: %d", o.ID, o.AccountsNumberFact)
	return &o, nil
}

// UpdateOrderAccountsNumber изменяет количество аккаунтов в заказе
func (db *DB) UpdateOrderAccountsNumber(orderID, newNumber int) (*models.Order, error) {
	tx, err := db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var o models.Order
	err = tx.QueryRow(
		`SELECT id, name, url, accounts_number_theory, accounts_number_fact, date_time FROM orders WHERE id = $1`,
		orderID,
	).Scan(&o.ID, &o.Name, &o.URL, &o.AccountsNumberTheory, &o.AccountsNumberFact, &o.DateTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	if _, err := tx.Exec(`UPDATE orders SET accounts_number_theory = $1 WHERE id = $2`, newNumber, orderID); err != nil {
		return nil, err
	}
	o.AccountsNumberTheory = newNumber

	if newNumber > o.AccountsNumberFact {
		// Добавляем недостающие аккаунты
		diff := newNumber - o.AccountsNumberFact
		rows, err := tx.Query(`SELECT id FROM accounts WHERE order_id IS NULL ORDER BY RANDOM() LIMIT $1`, diff)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		added := 0
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(`UPDATE accounts SET order_id = $1 WHERE id = $2`, orderID, id); err != nil {
				return nil, err
			}
			added++
		}
		o.AccountsNumberFact += added
	} else if newNumber < o.AccountsNumberFact {
		// Освобождаем лишние аккаунты
		diff := o.AccountsNumberFact - newNumber
		rows, err := tx.Query(`SELECT id FROM accounts WHERE order_id = $1 ORDER BY RANDOM() LIMIT $2`, orderID, diff)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		removed := 0
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(`UPDATE accounts SET order_id = NULL WHERE id = $1`, id); err != nil {
				return nil, err
			}
			removed++
		}
		o.AccountsNumberFact -= removed
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	log.Printf("[DB INFO] Заказ %d обновлён, фактических аккаунтов: %d", o.ID, o.AccountsNumberFact)
	return &o, nil
}

// GetOrderByID возвращает заказ по его идентификатору
// Используется для получения ссылки при обновлении описаний аккаунтов
func (db *DB) GetOrderByID(id int) (*models.Order, error) {
	var o models.Order
	err := db.Conn.QueryRow(
		`SELECT id, name, url, accounts_number_theory, accounts_number_fact, date_time FROM orders WHERE id = $1`,
		id,
	).Scan(&o.ID, &o.Name, &o.URL, &o.AccountsNumberTheory, &o.AccountsNumberFact, &o.DateTime)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
