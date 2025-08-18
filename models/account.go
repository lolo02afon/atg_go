package models

type Account struct {
	ID            int    `json:"id"`
	Phone         string `json:"phone"`
	ApiID         int    `json:"api_id"`
	ApiHash       string `json:"api_hash"`
	IsAuthorized  bool   `json:"is_authorized"`
	Gender        string `json:"gender"` // Пол аккаунта: male, female или neutral
	PhoneCodeHash string `json:"phone_code_hash"`
	ProxyID       *int   `json:"proxy_id"`
	OrderID       *int   `json:"order_id"` // ID выполняемого заказа (NULL, если аккаунт свободен)
	Proxy         *Proxy `json:"proxy"`
}
