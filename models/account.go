package models

type Account struct {
	ID            int    `json:"id"`
	Phone         string `json:"phone"`
	ApiID         int    `json:"api_id"`
	ApiHash       string `json:"api_hash"`
	IsAuthorized  bool   `json:"is_authorized"`
	PhoneCodeHash string `json:"phone_code_hash"`
}
