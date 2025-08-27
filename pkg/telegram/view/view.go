package view

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	module "atg_go/pkg/telegram/module"
	accountmutex "atg_go/pkg/telegram/module/account_mutex"

	"github.com/gotd/td/tg"
)

// ViewPost открывает пост канала, чтобы увеличить счётчик просмотров.
func ViewPost(db *storage.DB, acc models.Account, postURL string) error {
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
		trimmed := strings.TrimPrefix(postURL, "https://t.me/")
		parts := strings.Split(trimmed, "/")
		if len(parts) != 2 {
			return fmt.Errorf("некорректная ссылка на пост")
		}
		username := parts[0]
		msgID, err := strconv.Atoi(parts[1])
		if err != nil {
			return err
		}
		resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
		if err != nil {
			return err
		}
		ch, err := module.Modf_FindChannel(resolved.GetChats())
		if err != nil {
			return err
		}
		// Запрашиваем просмотр сообщения с флагом Increment=true
		_, err = api.MessagesGetMessagesViews(ctx, &tg.MessagesGetMessagesViewsRequest{
			Peer:      &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash},
			ID:        []int{msgID},
			Increment: true,
		})
		return err
	})
}
