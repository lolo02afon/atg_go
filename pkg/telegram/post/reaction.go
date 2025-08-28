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
	module "atg_go/pkg/telegram/module"
	accountmutex "atg_go/pkg/telegram/module/account_mutex"

	"github.com/gotd/td/tg"
)

// reactions содержит список базовых реакций, которые пытаемся использовать.
var reactions = []string{"❤️", "👍"}

// SendReaction добавляет реакцию к посту канала по ссылке postURL.
// Функция не фиксирует активность аккаунта.
func SendReaction(db *storage.DB, acc models.Account, postURL string) error {
	// Блокируем аккаунт на время операции, чтобы избежать параллельного использования
	if err := accountmutex.LockAccount(acc.ID); err != nil {
		return err
	}
	defer accountmutex.UnlockAccount(acc.ID)

	// Инициализируем клиента Telegram для указанного аккаунта
	client, err := module.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID, nil)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	return client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)

		// Извлекаем username канала и ID сообщения из ссылки вида https://t.me/name/id
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

		// Получаем информацию о канале по username
		resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
		if err != nil {
			return err
		}
		ch, err := module.Modf_FindChannel(resolved.GetChats())
		if err != nil {
			return err
		}

		// Пытаемся подписаться на канал; игнорируем ошибку, если уже участник
		_ = module.Modf_JoinChannel(ctx, api, ch, db, acc.ID)

		// Получаем разрешённые реакции для канала
		allowed, err := module.GetAllowedReactions(ctx, api, ch, reactions)
		if err != nil {
			return err
		}
		if len(allowed) == 0 {
			return fmt.Errorf("нет доступных реакций")
		}

		// Выбираем случайную реакцию из разрешённых и отправляем её
		reaction := allowed[rand.Intn(len(allowed))]
		_, err = api.MessagesSendReaction(ctx, &tg.MessagesSendReactionRequest{
			Peer:        &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash},
			MsgID:       msgID,
			Reaction:    []tg.ReactionClass{&tg.ReactionEmoji{Emoticon: reaction}},
			AddToRecent: true,
		})
		return err
	})
}
