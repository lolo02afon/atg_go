package telegram

import (
	"context"
	"fmt"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	module "atg_go/pkg/telegram/module"

	"github.com/gotd/td/tg"
)

// GetUserID возвращает ID пользователя Telegram для указанного аккаунта
func GetUserID(db *storage.DB, accountID int, phone string, apiID int, apiHash string, proxy *models.Proxy) (int, error) {
	client, err := module.Modf_AccountInitialization(apiID, apiHash, phone, proxy, nil, db.Conn, accountID)
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var userID int
	err = client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)
		full, err := api.UsersGetFullUser(ctx, &tg.InputUserSelf{})
		if err != nil {
			return fmt.Errorf("не удалось получить информацию о пользователе: %w", err)
		}
		userID = int(full.FullUser.ID)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return userID, nil
}
