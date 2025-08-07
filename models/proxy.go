package models

import "database/sql"

type Proxy struct {
	ID            int          `json:"id"`
	IP            string       `json:"ip"`
	Port          int          `json:"port"`
	Login         string       `json:"login"`
	Password      string       `json:"password"`
	IPv6          string       `json:"ipv6"`
	AccountsCount int          `json:"accounts_count"`
	IsActive      sql.NullBool `json:"is_active"`
}
