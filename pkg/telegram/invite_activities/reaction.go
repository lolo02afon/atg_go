// Package invite_activities содержит логику отправки реакций.
// Подпакет отделяет работу с реакциями от других частей системы.
package invite_activities

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	module "atg_go/pkg/telegram/technical"
	accountmutex "atg_go/pkg/telegram/technical/account_mutex"

	"github.com/gotd/td/tg"
)

// sendReaction добавляет реакцию к последнему сообщению обсуждения канала.
// В качестве идентификатора используется ID сообщения из чата обсуждения,
// который отличается от ID поста в канале. После успешной отправки
// сохраняет запись об активности в таблице activity. Возвращает ID сообщения,
// к которому была поставлена реакция (int), ID исходного канала (int) и
// ошибку. При неудаче оба идентификатора равны 0.
func sendReaction(db *storage.DB, accountID int, phone, channelURL string, apiID int, apiHash string, msgCount int, proxy *models.Proxy) (int, int, error) {
	log.Printf("[START] Отправка реакции в канал %s от имени %s", channelURL, phone)

	// Захватываем мьютекс для аккаунта, чтобы исключить параллельное использование
	if err := accountmutex.LockAccount(accountID); err != nil {
		return 0, 0, err
	}
	// Освобождаем мьютекс по завершении работы
	defer accountmutex.UnlockAccount(accountID)

	username, err := module.Modf_ExtractUsername(channelURL)
	if err != nil {
		// Возвращаем нулевые идентификаторы при ошибке извлечения username
		return 0, 0, fmt.Errorf("не удалось извлечь имя пользователя: %w", err)
	}

	client, err := module.Modf_AccountInitialization(apiID, apiHash, phone, proxy, nil, db.Conn, accountID, nil)
	if err != nil {
		// При ошибке инициализации возвращаем нулевые идентификаторы
		return 0, 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var (
		reactedMsgID int
		channelID    int
	)

	err = client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)

		resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
		if err != nil {
			if strings.Contains(err.Error(), "USERNAME_NOT_OCCUPIED") {
				_ = db.SaveCategoryChannelDelete(channelURL, models.ReasonChannelMissing)
			}
			return fmt.Errorf("не удалось распознать канал: %w", err)
		}

		// Находим сам канал по username
		channel, err := module.Modf_FindChannel(resolved.GetChats())
		if err != nil {
			_ = db.SaveCategoryChannelDelete(channelURL, models.ReasonChannelMissing)
			return err
		}
		channelID = int(channel.ID)

		// Пытаемся вступить в канал, чтобы иметь доступ к обсуждению
		if errJoin := module.Modf_JoinChannel(ctx, api, channel, db, accountID); errJoin != nil {
			if tg.IsChannelPrivate(errJoin) || strings.Contains(errJoin.Error(), "CHANNEL_PRIVATE") {
				_ = db.SaveCategoryChannelDelete(channelURL, models.ReasonChannelClosed)
				return errJoin
			}
			log.Printf("[ERROR] Не удалось вступить в канал: ID=%d Ошибка=%v", channel.ID, errJoin)
		}

		// Получаем чат обсуждения, не завязанный на конкретный пост
		discussionChat, err := module.Modf_getDiscussionChat(ctx, api, channel, db, accountID)
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "нет чата обсуждения") || strings.Contains(errStr, "не удалось найти чат обсуждения") {
				_ = db.SaveCategoryChannelDelete(channelURL, models.ReasonDiscussionClosed)
			} else if strings.Contains(errStr, "CHANNEL_PRIVATE") {
				_ = db.SaveCategoryChannelDelete(channelURL, models.ReasonChannelClosed)
			}
			return fmt.Errorf("не удалось получить чат обсуждения: %w", err)
		}

		if errJoin := module.Modf_JoinChannel(ctx, api, discussionChat, db, accountID); errJoin != nil {
			if tg.IsChannelPrivate(errJoin) || strings.Contains(errJoin.Error(), "CHANNEL_PRIVATE") {
				_ = db.SaveCategoryChannelDelete(channelURL, models.ReasonDiscussionClosed)
				return errJoin
			}
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

		// Запрашиваем последние сообщения из обсуждения
		messages, err := module.GetChannelPosts(ctx, api, discussionChat, msgCount)
		if err != nil {
			return fmt.Errorf("не удалось получить сообщения обсуждения: %w", err)
		}

		// Определяем сообщение, которому нужно поставить реакцию
		targetMsg, err := selectTargetMessage(messages, db, accountID, channelID)
		if err != nil {
			return err
		}

		// Проверяем, не слишком ли близко текущее сообщение к предыдущему
		// Передаём ID аккаунта, канала и сообщения
		canReact, err := db.CanReactOnMessage(accountID, channelID, targetMsg.ID)
		if err != nil {
			return fmt.Errorf("не удалось проверить возможность реакции: %w", err)
		}
		if !canReact {
			return fmt.Errorf("реакция на сообщение ID=%d запрещена: разница в ID должна быть не менее 10", targetMsg.ID)
		}

		// Выбираем реакцию: случайную из нашего списка, но если она не разрешена,
		// используем первую разрешённую админами обсуждения.
		reaction := pickReaction(reactionList, allowedReactions)

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
				if errSend = send(reaction); errSend != nil {
					return fmt.Errorf("не удалось отправить реакцию: %w", errSend)
				}
			} else {
				return fmt.Errorf("не удалось отправить реакцию: %w", errSend)
			}
		}

		log.Printf("Реакция %s успешно отправлена", reaction)
		// Сохраняем ID сообщения обсуждения (не ID поста канала)
		reactedMsgID = targetMsg.ID
		// Записываем активность в таблицу activity, используя ID сообщения обсуждения
		if err := module.SaveReactionActivity(db, accountID, channelID, reactedMsgID); err != nil {
			return fmt.Errorf("не удалось сохранить активность: %w", err)
		}

		return nil
	})

	return reactedMsgID, channelID, err
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

// selectTargetMessage выбирает самое новое сообщение без реакций,
// удовлетворяющее ограничению по минимальной разнице ID с последней
// реакцией аккаунта в данном канале.
// MessagesGetHistory возвращает сообщения от новых к старым, поэтому
// последовательно проверяем каждое сообщение. Если подходящее не найдено,
// возвращаем ошибку.
func selectTargetMessage(messages []*tg.Message, db *storage.DB, accountID, channelID int) (*tg.Message, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("нет сообщений для реакции")
	}
	for _, m := range messages {
		if len(m.Reactions.Results) != 0 {
			continue
		}
		// Проверяем возможность реакции с учётом последнего ID аккаунта
		// Передаём ID аккаунта, канала и сообщения
		canReact, err := db.CanReactOnMessage(accountID, channelID, m.ID)
		if err != nil {
			return nil, err
		}
		if canReact {
			return m, nil
		}
	}
	return nil, fmt.Errorf("подходящее сообщение без реакций не найдено")
}
