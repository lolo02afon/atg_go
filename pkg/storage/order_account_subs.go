package storage

import "atg_go/models"

// CountOrderSubs возвращает количество подписок аккаунтов на конкретный заказ.
func (db *DB) CountOrderSubs(orderID int) (int, error) {
	var count int
	err := db.Conn.QueryRow(`SELECT COUNT(*) FROM order_account_subs WHERE order_id = $1`, orderID).Scan(&count)
	return count, err
}

// AddOrderAccountSub сохраняет факт подписки аккаунта на канал заказа.
func (db *DB) AddOrderAccountSub(orderID, accountID int) error {
	_, err := db.Conn.Exec(`INSERT INTO order_account_subs (order_id, account_id) VALUES ($1, $2)`, orderID, accountID)
	return err
}

// GetRandomAccountsForOrder выбирает случайные авторизованные аккаунты,
// которые ещё не подписаны на канал указанного заказа и не помечены как мониторинговые.
func (db *DB) GetRandomAccountsForOrder(orderID, limit int) ([]models.Account, error) {
	condition := `a.is_authorized = true AND a.account_monitoring = false AND NOT EXISTS (
        SELECT 1 FROM order_account_subs oas WHERE oas.account_id = a.id AND oas.order_id = $1
    ) ORDER BY RANDOM() LIMIT $2`
	return db.getAccounts(condition, orderID, limit)
}
