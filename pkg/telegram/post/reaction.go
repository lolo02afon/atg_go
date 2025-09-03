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

// SendReaction добавляет реакцию к посту канала по ссылке postURL.
// orderID нужен для выбора предопределённых реакций из заказа (orders).
// Функция не фиксирует активность аккаунта.
func SendReaction(db *storage.DB, acc models.Account, orderID int, postURL string) error {
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

		// Получаем список реакций, разрешённых каналом (не более четырёх)
		suggested, err := module.GetChannelReactions(ctx, api, ch, 4)
		if err != nil {
			return err
		}

		// Запрашиваем сообщение, чтобы узнать уже поставленные реакции
		history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:     &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash},
			OffsetID: msgID + 1,
			Limit:    1,
		})
		if err != nil {
			return err
		}
		channelMessages, ok := history.(*tg.MessagesChannelMessages)
		if !ok || len(channelMessages.Messages) == 0 {
			return fmt.Errorf("сообщение не найдено")
		}
		msg, ok := channelMessages.Messages[0].(*tg.Message)
		if !ok {
			return fmt.Errorf("неподдерживаемый тип сообщения")
		}

		// Собираем реакции, уже присутствующие у поста
		var existing []string
		for _, rc := range msg.Reactions.Results {
			if emoji, ok := rc.Reaction.(*tg.ReactionEmoji); ok {
				existing = append(existing, emoji.Emoticon)
			}
		}

		// Получаем список реакций из заказа, если он задан
		reactions, err := db.GetPostReactionsForOrder(orderID)
		if err != nil {
			return err
		}

		rand.Seed(time.Now().UnixNano())

		if len(reactions) > 0 {
			// Используем заданную реакцию из БД
			reaction := reactions[rand.Intn(len(reactions))]
			send := func(r string) error {
				_, err = api.MessagesSendReaction(ctx, &tg.MessagesSendReactionRequest{
					Peer:        &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash},
					MsgID:       msgID,
					Reaction:    []tg.ReactionClass{&tg.ReactionEmoji{Emoticon: r}},
					AddToRecent: true,
				})
				return err
			}
			if err = send(reaction); err != nil && len(existing) > 0 {
				// При ошибке пробуем любую уже установленную реакцию
				reaction = existing[rand.Intn(len(existing))]
				err = send(reaction)
			}
			return err
		}

		// Стандартная логика выбора реакции
		candidates := suggested
		if len(existing) >= 2 {
			candidates = existing
		}
		if len(candidates) == 0 {
			return fmt.Errorf("нет доступных реакций")
		}
		reaction := candidates[rand.Intn(len(candidates))]

		_, err = api.MessagesSendReaction(ctx, &tg.MessagesSendReactionRequest{
			Peer:        &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash},
			MsgID:       msgID,
			Reaction:    []tg.ReactionClass{&tg.ReactionEmoji{Emoticon: reaction}},
			AddToRecent: true,
		})
		return err
	})
}
