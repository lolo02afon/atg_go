package telegram

import (
	"context"
	"fmt"
	"time"

	module "atg_go/pkg/telegram/module"

	"github.com/gotd/td/tg"
)

// GetUserID возвращает ID пользователя Telegram для указанного аккаунта
func GetUserID(phone string, apiID int, apiHash string) (int, error) {
	client, err := module.Modf_AccountInitialization(apiID, apiHash, phone)
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var userID int
	err = client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)
		full, err := api.UsersGetFullUser(ctx, &tg.UsersGetFullUserRequest{ID: &tg.InputUserSelf{}})
		if err != nil {
			return fmt.Errorf("не удалось получить информацию о пользователе: %w", err)
		}
		user, ok := full.User.(*tg.User)
		if !ok {
			return fmt.Errorf("некорректный тип пользователя")
		}
		userID = user.ID
		return nil
	})
	if err != nil {
		return 0, err
	}
	return userID, nil
}
