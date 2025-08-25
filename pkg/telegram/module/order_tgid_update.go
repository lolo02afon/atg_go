package module

import (
	"context"
	"fmt"
	"log"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	accountmutex "atg_go/pkg/telegram/module/account_mutex"

	"github.com/gotd/td/tg"
)

// Modf_UpdateOrdersChannelTGID заполняет channel_tgid для заказов, где оно отсутствует.
// Сначала пробует извлечь ID из ссылки, затем обращается к Telegram по username.
func Modf_UpdateOrdersChannelTGID(db *storage.DB) error {
	orders, err := db.GetOrdersWithoutChannelTGID()
	if err != nil {
		return err
	}
	if len(orders) == 0 {
		return nil
	}

	var pending []models.Order
	for _, o := range orders {
		if id := storage.ExtractChannelTGID(o.URLDefault); id != nil {
			if err := db.SetOrderChannelTGID(o.ID, *id); err != nil {
				return err
			}
		} else {
			pending = append(pending, o)
		}
	}
	if len(pending) == 0 {
		return nil
	}

	accounts, err := db.GetAuthorizedAccounts()
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		return fmt.Errorf("нет авторизованных аккаунтов для определения channel_tgid")
	}
	acc := accounts[0]

	if err := accountmutex.LockAccount(acc.ID); err != nil {
		return err
	}
	defer accountmutex.UnlockAccount(acc.ID)

	client, err := Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)
		for _, o := range pending {
			username, err := Modf_ExtractUsername(o.URLDefault)
			if err != nil {
				log.Printf("[WARN] некорректная ссылка %s для заказа %d", o.URLDefault, o.ID)
				continue
			}
			resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
			if err != nil {
				log.Printf("[WARN] не удалось определить канал %s: %v", o.URLDefault, err)
				continue
			}
			channel, err := Modf_FindChannel(resolved.GetChats())
			if err != nil {
				log.Printf("[WARN] канал %s не найден: %v", o.URLDefault, err)
				continue
			}
			if err := db.SetOrderChannelTGID(o.ID, fmt.Sprintf("%d", channel.ID)); err != nil {
				return err
			}
		}
		return nil
	})
}
