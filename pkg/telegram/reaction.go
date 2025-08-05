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
// канала. Возвращает ID сообщения, к которому была добавлена реакция.
func SendReaction(phone, channelURL string, apiID int, apiHash string, msgCount int) (int, error) {
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

		// Находим сам канал по username
		channel, err := module.Modf_FindChannel(resolved.GetChats())
		if err != nil {
			return err
		}
		log.Printf("[DEBUG] Найден канал ID=%d", channel.ID)

		// Пытаемся вступить в канал, чтобы иметь доступ к обсуждению
		if errJoin := module.Modf_JoinChannel(ctx, api, channel); errJoin != nil {
			log.Printf("[ERROR] Не удалось вступить в канал: ID=%d Ошибка=%v", channel.ID, errJoin)
		}

		// Получаем чат обсуждения, не завязанный на конкретный пост
		discussionChat, err := module.Modf_getDiscussionChat(ctx, api, channel)
		if err != nil {
			return fmt.Errorf("не удалось получить чат обсуждения: %w", err)
		}
		log.Printf("[DEBUG] Чат обсуждения ID=%d", discussionChat.ID)

		if errJoin := module.Modf_JoinChannel(ctx, api, discussionChat); errJoin != nil {
			log.Printf("[ERROR] Не удалось вступить в чат обсуждения: ID=%d Ошибка=%v", discussionChat.ID, errJoin)
		}

		// Получаем список разрешённых реакций
		allowedReactions, err := module.GetAllowedReactions(ctx, api, discussionChat, reactionList)
		if err != nil {
			return fmt.Errorf("не удалось получить доступные реакции: %w", err)
		}
		if len(allowedReactions) == 0 {
			return fmt.Errorf("нет доступных реакций в чате")
		}
		log.Printf("[DEBUG] Доступные реакции: %v", allowedReactions)

		// Запрашиваем последние сообщения из обсуждения
		messages, err := module.GetChannelPosts(ctx, api, discussionChat, msgCount)
		if err != nil {
			return fmt.Errorf("не удалось получить сообщения обсуждения: %w", err)
		}
		log.Printf("[DEBUG] Получено %d сообщений", len(messages))

		// Определяем сообщение, которому нужно поставить реакцию
		targetMsg, err := selectTargetMessage(messages)
		if err != nil {
			return err
		}
		log.Printf("[DEBUG] Целевое сообщение ID=%d", targetMsg.ID)

		// Ставим реакцию на найденное сообщение.
		// Если выбранная реакция недопустима, пробуем остальные.
		for len(allowedReactions) > 0 {
			reaction := getRandomReaction(allowedReactions)
			log.Printf("[DEBUG] Отправляем реакцию %s", reaction)
			_, err = api.MessagesSendReaction(ctx, &tg.MessagesSendReactionRequest{
				Peer:        &tg.InputPeerChannel{ChannelID: discussionChat.ID, AccessHash: discussionChat.AccessHash},
				MsgID:       targetMsg.ID,
				Reaction:    []tg.ReactionClass{&tg.ReactionEmoji{Emoticon: reaction}},
				AddToRecent: true,
			})
			if err == nil {
				reactedMsgID = targetMsg.ID
				log.Printf("Реакция %s успешно отправлена", reaction)
				return nil
			}
			if tg.IsReactionInvalid(err) {
				log.Printf("[WARN] Реакция %s недопустима: %v", reaction, err)
				allowedReactions = removeReaction(allowedReactions, reaction)
				continue
			}
			return fmt.Errorf("не удалось отправить реакцию: %w", err)
		}
		return fmt.Errorf("не удалось отправить ни одну допустимую реакцию")
	})

	return reactedMsgID, err
}

var reactionList = []string{"❤️", "😂"}

// getRandomReaction возвращает случайную реакцию из переданного списка.
func getRandomReaction(reactions []string) string {
	rand.Seed(time.Now().UnixNano())
	return reactions[rand.Intn(len(reactions))]
}

// removeReaction удаляет указанную реакцию из слайса.
func removeReaction(list []string, r string) []string {
	for i, v := range list {
		if v == r {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

// selectTargetMessage выбирает самое новое сообщение для отправки реакции.
// MessagesGetHistory возвращает сообщения в порядке от новых к старым,
// поэтому достаточно взять первый элемент списка.
// В противном случае возвращает ошибку.
func selectTargetMessage(messages []*tg.Message) (*tg.Message, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("нет сообщений для реакции")
	}
	return messages[0], nil
}
