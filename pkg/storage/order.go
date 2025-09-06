package storage

import (
	"atg_go/models"
	"database/sql"
	"log"
	"strings"

	"github.com/lib/pq"
)

// ExtractChannelTGID извлекает ID канала из ссылки вида https://t.me/c/<id>/...
// Возвращает nil, если ссылка не соответствует ожидаемому формату
func ExtractChannelTGID(url string) *string {
	const prefix = "https://t.me/c/"
	if strings.HasPrefix(url, prefix) {
		rest := strings.TrimPrefix(url, prefix)
		idPart := strings.SplitN(rest, "/", 2)[0]
		if idPart != "" {
			return &idPart
		}
	}
	return nil
}

// GetOrdersDefaultURLs возвращает список ссылок, от которых нельзя отписываться
func (db *DB) GetOrdersDefaultURLs() ([]string, error) {
	rows, err := db.Conn.Query(`SELECT url_default FROM orders`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var url sql.NullString
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		if url.Valid {
			urls = append(urls, url.String)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return urls, nil
}

// GetOrdersForMonitoring возвращает заказы с их ссылками, ID каналов, фактическим числом аккаунтов и числом активной аудитории.
// Эти данные нужны мониторинговым аккаунтам для подписки на каналы и расчёта метрик постов.
func (db *DB) GetOrdersForMonitoring() ([]models.Order, error) {
	rows, err := db.Conn.Query(`SELECT id, url_default, channel_tgid, accounts_number_fact, subs_active_count FROM orders WHERE url_default <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		var subsActiveCount sql.NullInt64 // временно читаем значение, чтобы обработать NULL
		if err := rows.Scan(&o.ID, &o.URLDefault, &o.ChannelTGID, &o.AccountsNumberFact, &subsActiveCount); err != nil {
			return nil, err
		}
		if subsActiveCount.Valid {
			val := int(subsActiveCount.Int64)
			o.SubsActiveCount = &val
		} else {
			o.SubsActiveCount = nil
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

// GetOrdersWithoutChannelTGID возвращает заказы без заполненного channel_tgid
// Используется перед обновлением описаний, чтобы знать, какие заказы требуют дополнения
func (db *DB) GetOrdersWithoutChannelTGID() ([]models.Order, error) {
	rows, err := db.Conn.Query(`SELECT id, url_default FROM orders WHERE channel_tgid IS NULL AND url_default <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.URLDefault); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

// SetOrderChannelTGID устанавливает значение channel_tgid для заказа
// Вызывается после получения ID канала по ссылке
func (db *DB) SetOrderChannelTGID(orderID int, channelTGID string) error {
	_, err := db.Conn.Exec(`UPDATE orders SET channel_tgid = $1 WHERE id = $2`, channelTGID, orderID)
	return err
}

// CreateOrder создаёт заказ и распределяет свободные аккаунты
func (db *DB) CreateOrder(o models.Order) (*models.Order, error) {
	tx, err := db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Проверяем указанные категории и игнорируем неизвестные.
	// Так заказ не привязывается к несуществующим категориям, но продолжает создаваться.
	if len(o.Category) > 0 {
		rows, err := tx.Query(`SELECT name FROM categories WHERE name = ANY($1)`, pq.Array(o.Category))
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		// Собираем найденные категории в множество для быстрого сравнения
		found := make(map[string]struct{})
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return nil, err
			}
			found[name] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}

		// Формируем итоговый список категорий и фиксируем отсутствующие
		var filtered []string
		for _, c := range o.Category {
			if _, ok := found[c]; ok {
				filtered = append(filtered, c)
			} else {
				// Предупреждаем, что категория не найдена, и продолжаем без неё
				log.Printf("категория %q не найдена", c)
			}
		}
		o.Category = filtered
	}

	// Фильтруем поле gender через общую функцию, чтобы хранить только допустимые значения
	gender := models.FilterGenders(o.Gender)

	// Извлекаем ID канала из ссылки по умолчанию, если это возможно
	channelTGID := ExtractChannelTGID(o.URLDefault)

	// Вставляем запись о заказе вместе со ссылкой по умолчанию и ID канала.
	// Если subs_active_count не указан, используем значение по умолчанию из БД.
	var (
		query           string
		args            []any
		subsActiveCount sql.NullInt64
	)

	// Передаём NULL, если после фильтрации категория отсутствует
	var categoriesArg any
	if len(o.Category) > 0 {
		categoriesArg = pq.Array(o.Category)
	} else {
		categoriesArg = nil
	}

	if o.SubsActiveCount != nil {
		query = `INSERT INTO orders (name, category, url_description, url_default, accounts_number_theory, gender, channel_tgid, subs_active_count) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, accounts_number_fact, date_time, subs_active_count`
		args = []any{o.Name, categoriesArg, o.URLDescription, o.URLDefault, o.AccountsNumberTheory, pq.Array(gender), channelTGID, *o.SubsActiveCount}
	} else {
		query = `INSERT INTO orders (name, category, url_description, url_default, accounts_number_theory, gender, channel_tgid) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, accounts_number_fact, date_time, subs_active_count`
		args = []any{o.Name, categoriesArg, o.URLDescription, o.URLDefault, o.AccountsNumberTheory, pq.Array(gender), channelTGID}
	}
	if err = tx.QueryRow(query, args...).Scan(&o.ID, &o.AccountsNumberFact, &o.DateTime, &subsActiveCount); err != nil {
		return nil, err
	}
	if subsActiveCount.Valid {
		val := int(subsActiveCount.Int64)
		o.SubsActiveCount = &val
	} else {
		o.SubsActiveCount = nil
	}
	o.Gender = gender
	o.ChannelTGID = channelTGID

	// Выбираем свободные аккаунты, исключая мониторинговые,
	// чтобы такие аккаунты не становились исполнителями заказов
	rows, err := tx.Query(
		`SELECT id FROM accounts WHERE order_id IS NULL AND account_monitoring = FALSE AND account_generator_category = FALSE ORDER BY RANDOM() LIMIT $1`,
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
	var subsActiveCount sql.NullInt64
	err = tx.QueryRow(
		// Приводим gender к text[], иначе pq не сможет сканировать массив enum
		`SELECT id, name, category, url_description, url_default, channel_tgid, accounts_number_theory, accounts_number_fact, subs_active_count, gender::text[], date_time FROM orders WHERE id = $1`,
		orderID,
	).Scan(&o.ID, &o.Name, &o.Category, &o.URLDescription, &o.URLDefault, &o.ChannelTGID, &o.AccountsNumberTheory, &o.AccountsNumberFact, &subsActiveCount, &o.Gender, &o.DateTime) // читаем текст, категории (pq.StringArray сканируется напрямую) и ссылку по умолчанию
	if subsActiveCount.Valid {
		val := int(subsActiveCount.Int64)
		o.SubsActiveCount = &val
	} else {
		o.SubsActiveCount = nil
	}
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
		// Добавляем недостающие аккаунты, игнорируя аккаунты под мониторингом
		diff := newNumber - o.AccountsNumberFact
		rows, err := tx.Query(`SELECT id FROM accounts WHERE order_id IS NULL AND account_monitoring = FALSE AND account_generator_category = FALSE ORDER BY RANDOM() LIMIT $1`, diff)
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
	var subsActiveCount sql.NullInt64
	err := db.Conn.QueryRow(
		// gender приводим к text[], чтобы избежать ошибок сканирования enum-массива
		`SELECT id, name, category, url_description, url_default, channel_tgid, accounts_number_theory, accounts_number_fact, subs_active_count, gender::text[], date_time FROM orders WHERE id = $1`,
		id,
	).Scan(&o.ID, &o.Name, &o.Category, &o.URLDescription, &o.URLDefault, &o.ChannelTGID, &o.AccountsNumberTheory, &o.AccountsNumberFact, &subsActiveCount, &o.Gender, &o.DateTime) // читаем текст, категории (pq.StringArray сканируется напрямую) и ссылку по умолчанию
	if subsActiveCount.Valid {
		val := int(subsActiveCount.Int64)
		o.SubsActiveCount = &val
	} else {
		o.SubsActiveCount = nil
	}
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

	// Получаем заказы; gender приводим к text[], чтобы pq корректно прочитал массив
	rows, err := tx.Query(`SELECT id, accounts_number_theory, accounts_number_fact, gender::text[] FROM orders`)
	if err != nil {
		log.Printf("[DB ERROR] выборка заказов: %v", err)
		return err
	}
	defer rows.Close()

	type orderData struct {
		id     int
		theory int
		fact   int
		gender pq.StringArray
	}
	var orders []orderData

	// Считываем все заказы из курсора
	for rows.Next() {
		var o orderData
		if err := rows.Scan(&o.id, &o.theory, &o.fact, &o.gender); err != nil {
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
		genders := o.gender

		// Сначала освобождаем аккаунты, чей пол не соответствует требованиям заказа
		misRows, err := tx.Query(
			`SELECT id FROM accounts WHERE order_id = $1 AND NOT (gender && $2::gender_enum[])`,
			orderID, pq.Array(genders),
		)
		if err != nil {
			log.Printf("[DB ERROR] выборка аккаунтов с неподходящим полом для заказа %d: %v", orderID, err)
			return err
		}
		var misIDs []int
		for misRows.Next() {
			var accID int
			if err := misRows.Scan(&accID); err != nil {
				misRows.Close()
				log.Printf("[DB ERROR] чтение аккаунта для очистки заказа %d: %v", orderID, err)
				return err
			}
			misIDs = append(misIDs, accID)
		}
		if err := misRows.Err(); err != nil {
			misRows.Close()
			log.Printf("[DB ERROR] курсор неподходящих аккаунтов для заказа %d: %v", orderID, err)
			return err
		}
		misRows.Close()
		for _, accID := range misIDs {
			if _, err := tx.Exec(`UPDATE accounts SET order_id = NULL WHERE id = $1`, accID); err != nil {
				log.Printf("[DB ERROR] освобождение аккаунта %d для заказа %d: %v", accID, orderID, err)
				return err
			}
		}

		// Считаем реальное количество аккаунтов, закреплённых за заказом, после очистки
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
					`SELECT id FROM accounts WHERE order_id IS NULL AND is_authorized = TRUE AND account_monitoring = FALSE AND account_generator_category = FALSE AND gender && $1::gender_enum[] ORDER BY RANDOM() LIMIT $2`,
					pq.Array(genders), need,
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
