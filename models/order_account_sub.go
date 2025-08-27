package models

// OrderAccountSub связывает аккаунт с заказом, на канал которого он подписан.
type OrderAccountSub struct {
	ID        int `json:"id"`         // Уникальный идентификатор записи
	OrderID   int `json:"order_id"`   // ID заказа, на канал которого подписался аккаунт
	AccountID int `json:"account_id"` // ID аккаунта, подписанного на канал
}
