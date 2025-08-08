package telegram

import (
	module "atg_go/pkg/telegram/module"
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"atg_go/models"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

type AuthHelper struct {
	phone         string
	code          string
	phoneCodeHash string
}

// SignUp реализует auth.UserAuthenticator (для новых регистраций)
func (a AuthHelper) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("sign up not implemented")
}

func (a AuthHelper) Phone(ctx context.Context) (string, error) {
	return a.phone, nil
}

func (a AuthHelper) Password(ctx context.Context) (string, error) {
	// Используем фиксированный пароль для аккаунтов с включенной 2FA
	return "Avgust134", nil
}

func (a AuthHelper) Code(ctx context.Context, _ *tg.AuthSentCode) (string, error) {
	return a.code, nil
}

func (a AuthHelper) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return nil
}

func RequestCode(apiID int, apiHash, phone string, proxy *models.Proxy) (string, error) {
	client, err := module.Modf_AccountInitialization(apiID, apiHash, phone, proxy, nil)
	if err != nil {
		return "", err
	}
	var phoneCodeHash string
	ctx := context.Background()
	err = client.Run(ctx, func(ctx context.Context) error {
		sentCode, err := client.Auth().SendCode(ctx, phone, auth.SendCodeOptions{})
		if err != nil {
			return err
		}
		if sentCode, ok := sentCode.(*tg.AuthSentCode); ok {
			phoneCodeHash = sentCode.PhoneCodeHash
			log.Printf("[DEBUG] Received phone_code_hash: %s", phoneCodeHash)
			log.Printf("[DEBUG] SentCode details: Type=%v, NextType=%v, Timeout=%v",
				sentCode.Type, sentCode.NextType, sentCode.Timeout)
		} else {
			log.Printf("[ERROR] Unexpected sent code type: %T", sentCode)
			return fmt.Errorf("unexpected sent code type: %T", sentCode)
		}
		return nil
	})
	return phoneCodeHash, err
}

func CompleteAuthorization(apiID int, apiHash, phone, code, phoneCodeHash string, proxy *models.Proxy) error {
	log.Printf("[DEBUG] Starting authorization for phone: %s", phone)

	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))
	client, err := module.Modf_AccountInitialization(apiID, apiHash, phone, proxy, randSrc)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return client.Run(ctx, func(ctx context.Context) error {
		helper := AuthHelper{
			phone:         phone,
			code:          code,
			phoneCodeHash: phoneCodeHash,
		}

		flow := auth.NewFlow(
			helper,
			auth.SendCodeOptions{},
		)

		log.Printf("[DEBUG] Attempting authorization with code: %s", code)
		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			log.Printf("[ERROR] Authorization failed: %v", err)
			return fmt.Errorf("authorization error: %w", err)
		}

		log.Printf("[INFO] Successfully authorized phone: %s", phone)
		return nil
	})
}
