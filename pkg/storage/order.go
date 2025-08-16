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

		var ids []int
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()

		for _, id := range ids {
			if _, err := tx.Exec(`UPDATE accounts SET order_id = $1 WHERE id = $2`, orderID, id); err != nil {
				return nil, err
			}
		}
		o.AccountsNumberFact += len(ids)
	} else if newNumber < o.AccountsNumberFact {
		// Освобождаем лишние аккаунты
		diff := o.AccountsNumberFact - newNumber
		rows, err := tx.Query(`SELECT id FROM accounts WHERE order_id = $1 ORDER BY RANDOM() LIMIT $2`, orderID, diff)
		if err != nil {
			return nil, err
		}

		var ids []int
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()

		for _, id := range ids {
			if _, err := tx.Exec(`UPDATE accounts SET order_id = NULL WHERE id = $1`, id); err != nil {
				return nil, err
			}
		}
		o.AccountsNumberFact -= len(ids)
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

// AssignFreeAccountsToOrders назначает свободные аккаунты заказам,
// у которых фактическое количество аккаунтов меньше требуемого.
// Комментарии на русском языке по требованию пользователя.
func (db *DB) AssignFreeAccountsToOrders() error {
	tx, err := db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Ищем заказы, где не хватает исполнителей
	rows, err := tx.Query(
		`SELECT id, accounts_number_theory, accounts_number_fact FROM orders WHERE accounts_number_fact < accounts_number_theory`,
	)
	if err != nil {
		return err
	}

	// Сначала собираем данные о нуждающихся в аккаунтах заказах,
	// чтобы не выполнять другие запросы, пока курсор не закрыт
	type need struct {
		id   int
		diff int
	}
	var needs []need
	for rows.Next() {
		var (
			id     int
			theory int
			fact   int
		)
		if err := rows.Scan(&id, &theory, &fact); err != nil {
			rows.Close()
			return err
		}
		needs = append(needs, need{id: id, diff: theory - fact})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	// Теперь можно закрыть курсор, чтобы освободить соединение
	rows.Close()

	// Для каждого заказа выделяем свободные аккаунты
	for _, n := range needs {
		accRows, err := tx.Query(
			`SELECT id FROM accounts WHERE order_id IS NULL AND is_authorized = TRUE ORDER BY RANDOM() LIMIT $1`,
			n.diff,
		)
		if err != nil {
			return err
		}

		// Сначала собираем идентификаторы свободных аккаунтов в срез
		var accIDs []int
		for accRows.Next() {
			var accID int
			if err := accRows.Scan(&accID); err != nil {
				accRows.Close()
				return err
			}
			accIDs = append(accIDs, accID)
		}
		if err := accRows.Err(); err != nil {
			accRows.Close()
			return err
		}
		// Курсор больше не нужен, закрываем перед выполнением обновлений
		accRows.Close()

		// Теперь обновляем аккаунты, назначая им order_id
		for _, accID := range accIDs {
			if _, err := tx.Exec(`UPDATE accounts SET order_id = $1 WHERE id = $2`, n.id, accID); err != nil {
				return err
			}
		}
		log.Printf("[DB INFO] Заказ %d, добавлено аккаунтов: %d", n.id, len(accIDs))
	}

	return tx.Commit()
}
