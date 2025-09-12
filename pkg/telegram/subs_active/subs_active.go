// Package subs_active отвечает за подписку аккаунтов на каналы заказов.
package subs_active

import (
	"context"
	"strings"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	module "atg_go/pkg/telegram/technical"
	accountmutex "atg_go/pkg/telegram/technical/account_mutex"

	"github.com/gotd/td/tg"
)

// SubscribeAccount подписывает аккаунт на канал заказа по ссылке.
func SubscribeAccount(db *storage.DB, acc models.Account, url string) error {
	// Блокируем аккаунт, чтобы исключить параллельное использование
	if err := accountmutex.LockAccount(acc.ID); err != nil {
		return err
	}
	defer accountmutex.UnlockAccount(acc.ID)

	client, err := module.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID, nil)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	return client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)
		username, err := module.Modf_ExtractUsername(url)
		if err != nil {
			return err
		}
		resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
		if err != nil {
			return err
		}
		channel, err := module.Modf_FindChannel(resolved.GetChats())
		if err != nil {
			return err
		}
		if err := module.Modf_JoinChannel(ctx, api, channel, db, acc.ID); err != nil {
			if !strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
				return err
			}
		}
		return nil
	})
}
