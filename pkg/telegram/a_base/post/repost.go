package post

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	module "atg_go/pkg/telegram/a_technical"
	accountmutex "atg_go/pkg/telegram/a_technical/account_mutex"

	"github.com/gotd/td/tg"
)

// SendRepost пересылает пост канала в "Избранное" аккаунта.
// Функция не фиксирует активность.
func SendRepost(db *storage.DB, acc models.Account, postURL string) error {
	// Защищаем аккаунт от параллельного использования
	if err := accountmutex.LockAccount(acc.ID); err != nil {
		return err
	}
	defer accountmutex.UnlockAccount(acc.ID)

	// Создаём клиента Telegram
	client, err := module.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID, nil)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	return client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)

		// Парсим ссылку поста
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

		// Разрешаем имя пользователя и находим канал
		resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
		if err != nil {
			return err
		}
		ch, err := module.Modf_FindChannel(resolved.GetChats())
		if err != nil {
			return err
		}

		// Подписываемся на канал при необходимости
		_ = module.Modf_JoinChannel(ctx, api, ch, db, acc.ID)

		// Пересылаем сообщение в личный чат (Избранное)
		_, err = api.MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
			FromPeer: &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash},
			ID:       []int{msgID},
			ToPeer:   &tg.InputPeerSelf{},
			RandomID: []int64{rand.Int63()},
		})
		return err
	})
}
