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

// SendComment - основная функция, которая:
// 1. Подключается к Telegram
// 2. Находит указанный канал
// 3. Выбирает случайный пост
// 4. Находит обсуждение этого поста
// 5. Отправляет случайный эмодзи в обсуждение
func SendComment(phone, channelURL string, apiID int, apiHash string, postsCount int) error {
	log.Printf("[START] Отправка эмодзи в канал %s от имени %s", channelURL, phone)

	// Извлекаем username из URL канала (например, из "https://t.me/channel" извлекаем "channel")
	username, err := module.Modf_ExtractUsername(channelURL)
	if err != nil {
		return fmt.Errorf("не удалось извлечь имя пользователя: %w", err)
	}

	// Создаем клиент Telegram с указанными параметрами
	client, err := module.Modf_AccountInitialization(apiID, apiHash, phone)
	if err != nil {
		return err
	}

	// Создаем контекст с таймаутом 60 секунд
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel() // Гарантируем отмену контекста при выходе из функции

	// Запускаем клиент и выполняем операции
	return client.Run(ctx, func(ctx context.Context) error {
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

		// 1️⃣ Запрашиваем пул постов один раз
		historyLimit := postsCount * 3
		if historyLimit < 10 {
			historyLimit = 10
		}
		posts, err := module.GetChannelPosts(ctx, api, channel, historyLimit)
		if err != nil {
			return fmt.Errorf("не удалось получить историю канала: %w", err)
		}

		// 2️⃣ Перемешиваем и проверяем первые postsCount
		rand.Shuffle(len(posts), func(i, j int) {
			posts[i], posts[j] = posts[j], posts[i]
		})

		var (
			discussionData *module.Discussion
			found          bool
		)

		//  Перемешали и будем искать первый пост, под которым нет комментариев наших ботов
		for i, p := range posts {
			if i >= postsCount {
				break
			}
			discussionData, err = module.Modf_getPostDiscussion(ctx, api, channel, p.ID)
			if err != nil {
				// просто пропускаем этот пост и идём дальше
				log.Printf("[DEBUG] пост %d: не удалось получить обсуждение (%v) — пропуск", p.ID, err)
				continue
			}

			// log.Printf("[DEBUG] выбран пост %d для комментирования", p.ID)

			found = true
			break
		}

		if !found {
			return fmt.Errorf("не удалось найти подходящий пост без комментариев после %d проверок", postsCount)
		}

		// всегда отвечаем именно на PostMessage из Discussion
		replyToMsgID := discussionData.PostMessage.ID

		// Подписываемся на группу обсуждения (в ней будут видны ответы)
		if errJoinDisc := module.Modf_JoinChannel(ctx, api, discussionData.Chat); errJoinDisc != nil {
			log.Printf("[ERROR] Не удалось присоединиться к чату обсуждений: ID=%d Ошибка=%v",
				discussionData.Chat.ID, errJoinDisc)
		}

		// Отправляем эмодзи
		return sendEmojiReply(ctx, api, &tg.InputPeerChannel{
			ChannelID:  discussionData.Chat.ID,
			AccessHash: discussionData.Chat.AccessHash,
		}, replyToMsgID)

	})
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
func sendEmojiReply(ctx context.Context, api *tg.Client, peer *tg.InputPeerChannel, replyToMsgID int) error {
	// Получаем случайный эмодзи
	emoji := getRandomEmoji()

	// Отправляем эмодзи как ответ (peer и replyToMsgID уже заданы вызывающим)
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
