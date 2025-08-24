package models

import "github.com/lib/pq"

type Account struct {
	ID                int            `json:"id"`
	Phone             string         `json:"phone"`
	ApiID             int            `json:"api_id"`
	ApiHash           string         `json:"api_hash"`
	IsAuthorized      bool           `json:"is_authorized"`
	AccountMonitoring bool           `json:"account_monitoring"` // Включает мониторинг аккаунта; такие аккаунты не назначаются на заказы
	Gender            pq.StringArray `json:"gender"`             // Пол(ы) аккаунта: допускается несколько значений
	PhoneCodeHash     string         `json:"phone_code_hash"`
	ProxyID           *int           `json:"proxy_id"`
	OrderID           *int           `json:"order_id"` // ID выполняемого заказа (NULL, если аккаунт свободен)
	Proxy             *Proxy         `json:"proxy"`
}
