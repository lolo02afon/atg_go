package telegram

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	module "atg_go/pkg/telegram/module"

	"github.com/gotd/td/tg"
)

// SendReaction выполняет добавление реакции к последнему сообщению обсуждения
// канала, у которого отсутствуют реакции. Возвращает ID сообщения, к которому
// была добавлена реакция.
func SendReaction(phone, channelURL string, apiID int, apiHash string, msgCount int, userIDs []int) (int, error) {
	log.Printf("[START] Отправка реакции в канал %s от имени %s", channelURL, phone)

	username, err := module.Modf_ExtractUsername(channelURL)
	if err != nil {
		return 0, fmt.Errorf("не удалось извлечь имя пользователя: %w", err)
	}

	client, err := module.Modf_AccountInitialization(apiID, apiHash, phone)
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var reactedMsgID int

	err = client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)

		resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
		if err != nil {
			return fmt.Errorf("не удалось распознать канал: %w", err)
		}

		channel, err := module.Modf_FindChannel(resolved.GetChats())
		if err != nil {
			return err
		}

		if errJoin := module.Modf_JoinChannel(ctx, api, channel); errJoin != nil {
			log.Printf("[ERROR] Не удалось вступить в канал: ID=%d Ошибка=%v", channel.ID, errJoin)
		}

		posts, err := module.GetChannelPosts(ctx, api, channel, 1)
		if err != nil {
			return fmt.Errorf("не удалось получить посты канала: %w", err)
		}

		discussion, err := module.Modf_getPostDiscussion(ctx, api, channel, posts[0].ID)
		if err != nil {
			return fmt.Errorf("не удалось получить обсуждение: %w", err)
		}

		if errJoin := module.Modf_JoinChannel(ctx, api, discussion.Chat); errJoin != nil {
			log.Printf("[ERROR] Не удалось вступить в чат обсуждения: ID=%d Ошибка=%v", discussion.Chat.ID, errJoin)
		}

		// Получаем список разрешённых реакций
		allowedReactions, err := module.GetAllowedReactions(ctx, api, discussion.Chat, reactionList)
		if err != nil {
			return fmt.Errorf("не удалось получить доступные реакции: %w", err)
		}
		if len(allowedReactions) == 0 {
			return fmt.Errorf("нет доступных реакций в чате")
		}

		// Запрашиваем последние сообщения из обсуждения поста
		messages, err := module.GetDiscussionReplies(ctx, api, discussion.Chat, discussion.PostMessage.ID, msgCount)
		if err != nil {
			return fmt.Errorf("не удалось получить сообщения обсуждения: %w", err)
		}

		idSet := make(map[int]struct{}, len(userIDs))
		for _, id := range userIDs {
			idSet[id] = struct{}{}
		}

		var lastUserMsg *tg.Message

		// Идём от последних сообщений к более ранним
		for i := len(messages) - 1; i >= 0; i-- {
			msg := messages[i]
			// Пропускаем сообщения без автора-пользователя (например, пост канала или служебные сообщения)
			from, ok := msg.FromID.(*tg.PeerUser)
			if !ok {
				continue
			}
			// Пропускаем сообщения наших аккаунтов
			if _, exists := idSet[int(from.UserID)]; exists {
				continue
			}

			// Сохраняем первое подходящее сообщение для fallback
			if lastUserMsg == nil {
				lastUserMsg = msg
			}

			// Отправляем реакцию, если у сообщения нет других реакций
			if len(msg.Reactions.Results) == 0 {
				reaction := getRandomReaction(allowedReactions)
				_, err := api.MessagesSendReaction(ctx, &tg.MessagesSendReactionRequest{
					Peer:        &tg.InputPeerChannel{ChannelID: discussion.Chat.ID, AccessHash: discussion.Chat.AccessHash},
					MsgID:       msg.ID,
					Reaction:    []tg.ReactionClass{&tg.ReactionEmoji{Emoticon: reaction}},
					AddToRecent: true,
				})
				if err != nil {
					return fmt.Errorf("не удалось отправить реакцию: %w", err)
				}
				reactedMsgID = msg.ID
				log.Printf("Реакция %s успешно отправлена", reaction)
				return nil
			}
		}

		// Если все сообщения уже имеют реакции, реагируем на первое подходящее
		if lastUserMsg != nil {
			reaction := getRandomReaction(allowedReactions)
			_, err := api.MessagesSendReaction(ctx, &tg.MessagesSendReactionRequest{
				Peer:        &tg.InputPeerChannel{ChannelID: discussion.Chat.ID, AccessHash: discussion.Chat.AccessHash},
				MsgID:       lastUserMsg.ID,
				Reaction:    []tg.ReactionClass{&tg.ReactionEmoji{Emoticon: reaction}},
				AddToRecent: true,
			})
			if err != nil {
				return fmt.Errorf("не удалось отправить реакцию: %w", err)
			}
			reactedMsgID = lastUserMsg.ID
			log.Printf("Реакция %s успешно отправлена", reaction)
			return nil
		}

		log.Printf("[INFO] Не найдено подходящих сообщений в обсуждении")
		return nil
	})

	return reactedMsgID, err
}

var reactionList = []string{"❤️", "😂"}

// getRandomReaction возвращает случайную реакцию из переданного списка.
func getRandomReaction(reactions []string) string {
	rand.Seed(time.Now().UnixNano())
	return reactions[rand.Intn(len(reactions))]
}
