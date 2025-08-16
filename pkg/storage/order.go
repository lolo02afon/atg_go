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

// AssignFreeAccountsToOrders синхронизирует аккаунты в заказах согласно требуемому количеству.
// Для каждого заказа добавляются недостающие аккаунты или освобождаются лишние.
func (db *DB) AssignFreeAccountsToOrders() error {
	tx, err := db.Conn.Begin()
	if err != nil {
		log.Printf("[DB ERROR] начало транзакции: %v", err)
		return err
	}
	defer tx.Rollback()

	// Получаем все заказы и загружаем их в память, чтобы затем закрыть курсор
	rows, err := tx.Query(`SELECT id, accounts_number_theory, accounts_number_fact FROM orders`)
	if err != nil {
		log.Printf("[DB ERROR] выборка заказов: %v", err)
		return err
	}
	defer rows.Close()

	type orderData struct {
		id     int
		theory int
		fact   int
	}
	var orders []orderData

	// Считываем все заказы из курсора
	for rows.Next() {
		var o orderData
		if err := rows.Scan(&o.id, &o.theory, &o.fact); err != nil {
			log.Printf("[DB ERROR] чтение заказа: %v", err)
			return err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[DB ERROR] курсор заказов: %v", err)
		return err
	}
	// Закрываем курсор до выполнения дальнейших запросов, чтобы освободить соединение
	rows.Close()

	// Обрабатываем каждый заказ по отдельности
	for _, o := range orders {
		orderID := o.id
		theory := o.theory
		fact := o.fact

		// Считаем реальное количество аккаунтов, закреплённых за заказом
		var actual int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM accounts WHERE order_id = $1`, orderID).Scan(&actual); err != nil {
			log.Printf("[DB ERROR] подсчёт аккаунтов заказа %d: %v", orderID, err)
			return err
		}

		// Если сохранённое значение не совпадает с реальным, обновляем accounts_number_fact
		if fact != actual {
			if _, err := tx.Exec(`UPDATE orders SET accounts_number_fact = $1 WHERE id = $2`, actual, orderID); err != nil {
				log.Printf("[DB ERROR] обновление фактического количества для заказа %d: %v", orderID, err)
				return err
			}
			fact = actual
		}

		// Если желаемое количество не совпадает с фактическим, корректируем аккаунты
		if theory != actual {
			if theory > actual {
				// Нужно добавить недостающие аккаунты
				need := theory - actual
				accRows, err := tx.Query(
					`SELECT id FROM accounts WHERE order_id IS NULL AND is_authorized = TRUE ORDER BY RANDOM() LIMIT $1`,
					need,
				)
				if err != nil {
					log.Printf("[DB ERROR] выборка аккаунтов для заказа %d: %v", orderID, err)
					return err
				}

				var accIDs []int
				for accRows.Next() {
					var accID int
					if err := accRows.Scan(&accID); err != nil {
						accRows.Close()
						log.Printf("[DB ERROR] чтение аккаунта для заказа %d: %v", orderID, err)
						return err
					}
					accIDs = append(accIDs, accID)
				}
				if err := accRows.Err(); err != nil {
					accRows.Close()
					log.Printf("[DB ERROR] курсор аккаунтов для заказа %d: %v", orderID, err)
					return err
				}
				accRows.Close()

				for _, accID := range accIDs {
					if _, err := tx.Exec(`UPDATE accounts SET order_id = $1 WHERE id = $2`, orderID, accID); err != nil {
						log.Printf("[DB ERROR] обновление аккаунта %d: %v", accID, err)
						return err
					}
				}
				actual += len(accIDs)
			} else {
				// Нужно освободить лишние аккаунты
				needRelease := actual - theory
				accRows, err := tx.Query(
					`SELECT id FROM accounts WHERE order_id = $1 ORDER BY RANDOM() LIMIT $2`,
					orderID, needRelease,
				)
				if err != nil {
					log.Printf("[DB ERROR] выборка аккаунтов для освобождения заказа %d: %v", orderID, err)
					return err
				}

				var accIDs []int
				for accRows.Next() {
					var accID int
					if err := accRows.Scan(&accID); err != nil {
						accRows.Close()
						log.Printf("[DB ERROR] чтение аккаунта для освобождения заказа %d: %v", orderID, err)
						return err
					}
					accIDs = append(accIDs, accID)
				}
				if err := accRows.Err(); err != nil {
					accRows.Close()
					log.Printf("[DB ERROR] курсор аккаунтов для освобождения заказа %d: %v", orderID, err)
					return err
				}
				accRows.Close()

				for _, accID := range accIDs {
					if _, err := tx.Exec(`UPDATE accounts SET order_id = NULL WHERE id = $1`, accID); err != nil {
						log.Printf("[DB ERROR] обновление аккаунта %d: %v", accID, err)
						return err
					}
				}
				actual -= len(accIDs)
			}

			// После изменений фиксируем фактическое количество аккаунтов
			if _, err := tx.Exec(`UPDATE orders SET accounts_number_fact = $1 WHERE id = $2`, actual, orderID); err != nil {
				log.Printf("[DB ERROR] обновление фактического количества для заказа %d: %v", orderID, err)
				return err
			}
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("[DB ERROR] курсор заказов: %v", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[DB ERROR] коммит транзакции: %v", err)
		return err
	}
	return nil
}

// DeleteOrder удаляет заказ по идентификатору
// При удалении благодаря ON DELETE SET NULL у связанных аккаунтов очищается поле order_id
func (db *DB) DeleteOrder(id int) error {
	if _, err := db.Conn.Exec(`DELETE FROM orders WHERE id = $1`, id); err != nil {
		return err
	}
	return nil
}
