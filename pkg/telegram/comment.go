package telegram

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"atg_go/pkg/storage"
	module "atg_go/pkg/telegram/module"

	"github.com/gotd/td/tg"
)

// SendComment подключается к Telegram, находит случайный пост в указанном канале
// и отправляет случайный эмодзи в обсуждение этого поста.
// После отправки сохраняет запись об активности в таблице activity.
// Возвращает ID поста, к которому оставлен комментарий (int),
// ID исходного канала (int) и ошибку.
// При неудаче оба идентификатора равны 0.
func SendComment(db *storage.DB, accountID int, phone, channelURL string, apiID int, apiHash string, postsCount int, canSend func(channelID, messageID int) (bool, error), userIDs []int) (int, int, error) {
	log.Printf("[START] Отправка эмодзи в канал %s от имени %s", channelURL, phone)

	// Извлекаем username из URL канала (например, из "https://t.me/channel" извлекаем "channel")
	username, err := module.Modf_ExtractUsername(channelURL)
	if err != nil {
		// Возвращаем нулевые значения для идентификаторов при ошибке
		return 0, 0, fmt.Errorf("не удалось извлечь имя пользователя: %w", err)
	}

	// Создаем клиент Telegram с указанными параметрами
	client, err := module.Modf_AccountInitialization(apiID, apiHash, phone)
	if err != nil {
		// При ошибке инициализации также возвращаем нулевые идентификаторы
		return 0, 0, err
	}

	// Создаем контекст с таймаутом 60 секунд
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel() // Гарантируем отмену контекста при выходе из функции

	var (
		msgID     int
		channelID int
	)

	// Запускаем клиент и выполняем операции
	err = client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client) // Создаем API-клиент

		// Получаем информацию о канале по username
		resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
			Username: username,
		})
		if err != nil {
			return fmt.Errorf("не удалось распознать канал: %w", err)
		}

		// Находим канал среди полученных чатов
		channel, err := module.Modf_FindChannel(resolved.GetChats())
		if err != nil {
			return err
		}

		// Подписываемся на сам канал, чтобы получить доступ к дискуссии
		if errJoinChannel := module.Modf_JoinChannel(ctx, api, channel); errJoinChannel != nil {
			log.Printf("[ERROR] Не удалось вступить в канал: ID=%d AccessHash=%d Ошибка=%v",
				channel.ID, channel.AccessHash, errJoinChannel)
		}

		// 1️⃣ Запрашиваем только последние postsCount постов
		// чтобы не уходить глубоко в историю канала
		if postsCount < 1 {
			postsCount = 1
		}
		posts, err := module.GetChannelPosts(ctx, api, channel, postsCount)
		if err != nil {
			return fmt.Errorf("не удалось получить историю канала: %w", err)
		}

		idSet := make(map[int]struct{}, len(userIDs))
		for _, id := range userIDs {
			idSet[id] = struct{}{}
		}

		for _, p := range posts {
			discussionData, err := module.Modf_getPostDiscussion(ctx, api, channel, p.ID)
			if err != nil {
				log.Printf("[DEBUG] пост %d: не удалось получить обсуждение (%v) — пропуск", p.ID, err)
				continue
			}

			// Присоединяемся к чату обсуждения, чтобы иметь возможность читать и оставлять комментарии
			if errJoinDisc := module.Modf_JoinChannel(ctx, api, discussionData.Chat); errJoinDisc != nil {
				log.Printf("[ERROR] Не удалось присоединиться к чату обсуждений: ID=%d Ошибка=%v", discussionData.Chat.ID, errJoinDisc)
			}

			replyToMsgID := discussionData.PostMessage.ID

			if canSend != nil {
				// Используем ID исходного канала при проверке возможности отправки
				allowed, err := canSend(int(channel.ID), replyToMsgID)
				if err != nil {
					return err
				}
				if !allowed {
					log.Printf("[INFO] Пост %d уже комментирован нашими аккаунтами, пропуск для %s", replyToMsgID, phone)
					continue
				}
			}

			hasOwn, err := hasRecentCommentByUsers(ctx, api, discussionData.Chat, replyToMsgID, idSet)
			if err != nil {
				log.Printf("[DEBUG] пост %d: не удалось проверить последние комментарии (%v) — пропуск", p.ID, err)
				continue
			}
			if hasOwn {
				log.Printf("[INFO] Пост %d уже комментирован нашими аккаунтами, пропуск для %s", replyToMsgID, phone)
				continue
			}

			// Отправляем эмодзи-ответ
			if err := sendEmojiReply(ctx, api, &tg.InputPeerChannel{
				ChannelID:  discussionData.Chat.ID,
				AccessHash: discussionData.Chat.AccessHash,
			}, replyToMsgID); err != nil {
				return err
			}

			// Сохраняем ID исходного поста
			msgID = replyToMsgID
			// Сохраняем ID канала, приводя его к типу int
			channelID = int(channel.ID)
			// Записываем активность в таблицу activity по ID поста
			if err := module.SaveCommentActivity(db, accountID, channelID, msgID); err != nil {
				return fmt.Errorf("не удалось сохранить активность: %w", err)
			}

			return nil
		}

		return fmt.Errorf("не удалось найти подходящий пост без комментариев после %d проверок", postsCount)

	})

	return msgID, channelID, err
}

var emojiList = []string{
	"🤡", "🥃", "🌶", "✊🏿", "👃🏿", "🦷", "👜", "👛", "👑", "🎚", "🏴", "🇰🇵",
	"🦧", "🦥", "🦄", "🦦", "🐷", "🐦",
	// Каска (повторяется три раза, чтобы увеличить шанс ее выбора)
	"🪖", "🪖", "🪖",
	"спасибо за пост) досвиданья",
	")", ")", ")",
}

// возвращает случайный элемент из emojiList
func getRandomEmoji() string {
	// Инициализация генератора случайных чисел текущим временем
	rand.Seed(time.Now().UnixNano())
	// Выбор случайного индекса и возврат соответствующего элемента
	return emojiList[rand.Intn(len(emojiList))]
}

// отправляет выбранный эмодзи как ответ на указанное сообщение
// при успешной отправке возвращает nil
func sendEmojiReply(ctx context.Context, api *tg.Client, peer *tg.InputPeerChannel, replyToMsgID int) error {
	// Получаем случайный эмодзи
	emoji := getRandomEmoji()

	// Отправляем эмодзи как ответ
	_, err := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
		Peer:     peer,
		Message:  emoji,
		ReplyTo:  &tg.InputReplyToMessage{ReplyToMsgID: replyToMsgID},
		RandomID: rand.Int63(),
	})
	if err != nil {
		return fmt.Errorf("не удалось отправить эмодзи: %w", err)
	}

	log.Printf("Эмодзи %s успешно отправлен", emoji)
	return nil
}

// проверяет, есть ли среди последних комментариев к посту сообщения от наших аккаунтов
func hasRecentCommentByUsers(ctx context.Context, api *tg.Client, chat *tg.Channel, msgID int, userIDs map[int]struct{}) (bool, error) {
	if len(userIDs) == 0 {
		return false, nil
	}

	res, err := api.MessagesGetReplies(ctx, &tg.MessagesGetRepliesRequest{
		Peer:  &tg.InputPeerChannel{ChannelID: chat.ID, AccessHash: chat.AccessHash},
		MsgID: msgID,
		Limit: 30,
	})
	if err != nil {
		return false, fmt.Errorf("не удалось получить комментарии: %w", err)
	}

	msgs, ok := res.(*tg.MessagesChannelMessages)
	if !ok {
		return false, fmt.Errorf("неожиданный тип ответа")
	}

	for _, m := range msgs.Messages {
		msg, ok := m.(*tg.Message)
		if !ok {
			continue
		}
		if from, ok := msg.FromID.(*tg.PeerUser); ok {
			if _, exist := userIDs[int(from.UserID)]; exist {
				return true, nil
			}
		}
	}

	return false, nil
}
