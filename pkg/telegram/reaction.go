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

		// Выбираем реакцию: случайную из нашего списка, но если она не разрешена,
		// используем первую разрешённую админами обсуждения.
		reaction := pickReaction(reactionList, allowedReactions)
		log.Printf("[DEBUG] Отправляем реакцию %s", reaction)

		send := func(r string) error {
			_, err = api.MessagesSendReaction(ctx, &tg.MessagesSendReactionRequest{
				Peer:        &tg.InputPeerChannel{ChannelID: discussionChat.ID, AccessHash: discussionChat.AccessHash},
				MsgID:       targetMsg.ID,
				Reaction:    []tg.ReactionClass{&tg.ReactionEmoji{Emoticon: r}},
				AddToRecent: true,
			})
			return err
		}

		if errSend := send(reaction); errSend != nil {
			if tg.IsReactionInvalid(errSend) && reaction != allowedReactions[0] {
				log.Printf("[WARN] Реакция %s недопустима: %v", reaction, errSend)
				reaction = allowedReactions[0]
				log.Printf("[DEBUG] Отправляем реакцию %s", reaction)
				if errSend = send(reaction); errSend != nil {
					return fmt.Errorf("не удалось отправить реакцию: %w", errSend)
				}
			} else {
				return fmt.Errorf("не удалось отправить реакцию: %w", errSend)
			}
		}

		reactedMsgID = targetMsg.ID
		log.Printf("Реакция %s успешно отправлена", reaction)
		return nil
	})

	return reactedMsgID, err
}

var reactionList = []string{"❤️", "👍"}

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

// getRandomReaction возвращает случайную реакцию из переданного списка.
func getRandomReaction(reactions []string) string {
	return reactions[rnd.Intn(len(reactions))]
}

// pickReaction выбирает случайную реакцию из base, если она разрешена.
// Если случайно выбранная реакция запрещена, возвращает первую из allowed.
func pickReaction(base, allowed []string) string {
	r := getRandomReaction(base)
	for _, a := range allowed {
		if r == a {
			return r
		}
	}
	return allowed[0]
}

// selectTargetMessage выбирает самое новое сообщение без реакций.
// MessagesGetHistory возвращает сообщения от новых к старым,
// поэтому проходим по списку и ищем первое сообщение,
// у которого отсутствуют реакции. Если все сообщения уже
// содержат реакции, возвращаем ошибку.
func selectTargetMessage(messages []*tg.Message) (*tg.Message, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("нет сообщений для реакции")
	}
	for _, m := range messages {
		if len(m.Reactions.Results) == 0 {
			return m, nil
		}
	}
	return nil, fmt.Errorf("подходящее сообщение без реакций не найдено")
}
