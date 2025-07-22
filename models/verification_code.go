package models

type VerificationCode struct {
	ID         int    `json:"id"`
	AccountID  int    `json:"account_id"`
	Code       string `json:"code"`
	IsVerified bool   `json:"is_verified"`
	Send       bool   `json:"send"`
}
