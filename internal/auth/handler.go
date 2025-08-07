package auth

import (
	"log"
	"strconv"

	"atg_go/models"
	"atg_go/pkg/storage"
	telegram "atg_go/pkg/telegram"

	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	DB *storage.DB
}

func NewHandler(db *storage.DB) *AccountHandler {
	return &AccountHandler{DB: db}
}

func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var account models.Account
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(400, gin.H{"error": "Invalid data"})
		return
	}

	var proxy *models.Proxy
	if account.ProxyID != nil {
		p, err := h.DB.GetProxyByID(*account.ProxyID)
		if err != nil {
			c.JSON(400, gin.H{"error": "Proxy not found"})
			return
		}
		if p.AccountsCount >= 30 {
			c.JSON(400, gin.H{"error": "Proxy limit reached"})
			return
		}
		proxy = p
	}

	hash, err := telegram.RequestCode(account.ApiID, account.ApiHash, account.Phone, proxy)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to request code"})
		return
	}

	account.PhoneCodeHash = hash
	created, err := h.DB.CreateAccount(account)
	if err != nil {
		c.JSON(500, gin.H{"error": "DB error"})
		return
	}

	c.JSON(200, gin.H{"id": created.ID})
}

func (h *AccountHandler) VerifyAccount(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var input struct {
		Code string `json:"code"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Invalid code"})
		return
	}

	account, err := h.DB.GetAccountByID(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "Account not found"})
		return
	}

	log.Printf("[DEBUG] VerifyAccount - Phone: %s, Code: %s, Hash: %s", account.Phone, input.Code, account.PhoneCodeHash)

	if err := telegram.CompleteAuthorization(
		account.ApiID,
		account.ApiHash,
		account.Phone,
		input.Code,
		account.PhoneCodeHash,
		account.Proxy,
	); err != nil {
		c.JSON(400, gin.H{"error": "Auth failed: " + err.Error()})
		return
	}

	// Помечаем аккаунт как авторизованный
	if err := h.DB.MarkAccountAsAuthorized(account.ID); err != nil {
		c.JSON(500, gin.H{"error": "Failed to mark account as authorized"})
		return
	}

	c.JSON(200, gin.H{"status": "Authorized!"})
}
