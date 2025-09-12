// Package auth содержит функции для авторизации аккаунтов в Telegram.
// Логика вынесена в отдельный подпакет, чтобы изолировать работу с входом.
package accounts_auth

import (
	module "atg_go/pkg/telegram/technical"
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

const twoFAPassword = "Avgust134"

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
	return twoFAPassword, nil
}

func (a AuthHelper) Code(ctx context.Context, _ *tg.AuthSentCode) (string, error) {
	return a.code, nil
}

func (a AuthHelper) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return nil
}

// RequestCode отправляет код подтверждения и сохраняет хеш в БД
func RequestCode(apiID int, apiHash, phone string, proxy *models.Proxy, db *storage.DB, accountID int) (string, error) {
	client, err := module.Modf_AccountInitialization(apiID, apiHash, phone, proxy, nil, db.Conn, accountID, nil)
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
		if sent, ok := sentCode.(*tg.AuthSentCode); ok {
			phoneCodeHash = sent.PhoneCodeHash
			// Сохраняем полученный хеш в БД для дальнейшей авторизации
			if err := db.UpdatePhoneCodeHash(accountID, phoneCodeHash); err != nil {
				return err
			}
		} else {
			log.Printf("[ERROR] Unexpected sent code type: %T", sentCode)
			return fmt.Errorf("unexpected sent code type: %T", sentCode)
		}
		return nil
	})
	return phoneCodeHash, err
}

func CompleteAuthorization(db *storage.DB, accountID, apiID int, apiHash, phone, code, phoneCodeHash string, proxy *models.Proxy) error {
	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))
	client, err := module.Modf_AccountInitialization(apiID, apiHash, phone, proxy, randSrc, db.Conn, accountID, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return client.Run(ctx, func(ctx context.Context) error {
		if _, err := client.Auth().SignIn(ctx, phone, code, phoneCodeHash); err != nil {
			if errors.Is(err, auth.ErrPasswordAuthNeeded) {
				if _, err := client.Auth().Password(ctx, twoFAPassword); err != nil {
					log.Printf("[ERROR] Password authentication failed: %v", err)
					return fmt.Errorf("password authentication failed: %w", err)
				}
				log.Printf("[INFO] Successfully authorized phone: %s", phone)
				return nil
			}
			log.Printf("[ERROR] Authorization failed: %v", err)
			return fmt.Errorf("authorization error: %w", err)
		}

		log.Printf("[INFO] Successfully authorized phone: %s", phone)
		return nil
	})
}
