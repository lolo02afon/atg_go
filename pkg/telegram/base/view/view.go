package view

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	module "atg_go/pkg/telegram/technical"
	accountmutex "atg_go/pkg/telegram/technical/account_mutex"

	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"
)

// ViewPost открывает пост канала, чтобы увеличить счётчик просмотров.
func ViewPost(db *storage.DB, acc models.Account, postURL string) error {
	// Инициализируем генератор случайных чисел для задержек просмотра
	rand.Seed(time.Now().UnixNano())

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

		// Получаем сообщение, чтобы определить тип содержимого
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
		message, ok := channelMessages.Messages[0].(*tg.Message)
		if !ok {
			return fmt.Errorf("неподдерживаемый тип сообщения")
		}
		delay := viewDelay(message)

		// Запрашиваем просмотр сообщения с флагом Increment=true
		_, err = api.MessagesGetMessagesViews(ctx, &tg.MessagesGetMessagesViewsRequest{
			Peer:      &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash},
			ID:        []int{msgID},
			Increment: true,
		})
		if err != nil {
			return err
		}

		// Открываем вложения для имитации реального просмотра
		if err := openMedia(ctx, api, message); err != nil {
			return err
		}

		// Имитация времени просмотра
		time.Sleep(delay)

		// Сохраняем факт просмотра в таблице активности
		return module.SaveViewActivity(db, acc.ID, int(ch.ID), msgID)
	})
}

// openMedia скачивает вложение сообщения для имитации открытия.
func openMedia(ctx context.Context, api *tg.Client, m *tg.Message) error {
	d := downloader.NewDownloader()
	switch media := m.Media.(type) {
	case *tg.MessageMediaPhoto:
		photo, ok := media.Photo.(*tg.Photo)
		if !ok {
			return nil
		}
		// Выбираем последнюю доступную размерность фотографии
		var size string
		for _, s := range photo.Sizes {
			if v, ok := s.(interface{ GetType() string }); ok {
				size = v.GetType()
			}
		}
		if size == "" {
			return nil
		}
		loc := &tg.InputPhotoFileLocation{
			ID:            photo.ID,
			AccessHash:    photo.AccessHash,
			FileReference: photo.FileReference,
			ThumbSize:     size,
		}
		return downloadToTemp(ctx, d, api, loc)
	case *tg.MessageMediaDocument:
		doc, ok := media.Document.(*tg.Document)
		if !ok {
			return nil
		}
		loc := &tg.InputDocumentFileLocation{
			ID:            doc.ID,
			AccessHash:    doc.AccessHash,
			FileReference: doc.FileReference,
			ThumbSize:     "",
		}
		return downloadToTemp(ctx, d, api, loc)
	default:
		return nil
	}
}

// downloadToTemp сохраняет вложение во временный файл и удаляет его после использования.
// loc описывает путь к файлу в Telegram.
func downloadToTemp(ctx context.Context, d *downloader.Downloader, api *tg.Client, loc tg.InputFileLocationClass) error {
	// Создаём временный файл в системной директории
	f, err := os.CreateTemp("", "tg-media-*")
	if err != nil {
		return err
	}
	// Закрываем и удаляем файл после завершения скачивания
	defer func() {
		_ = f.Close()
		os.Remove(f.Name())
	}()

	// Сохраняем содержимое в файл
	_, err = d.Download(api, loc).Stream(ctx, f)
	return err
}

// viewDelay выбирает задержку просмотра в зависимости от типа содержимого сообщения.
func viewDelay(m *tg.Message) time.Duration {
	switch media := m.Media.(type) {
	case *tg.MessageMediaPhoto:
		// Для изображений ждём 3–7 секунд
		return randomDuration(3, 7)
	case *tg.MessageMediaDocument:
		if doc, ok := media.Document.(*tg.Document); ok {
			for _, attr := range doc.Attributes {
				if v, ok := attr.(*tg.DocumentAttributeVideo); ok {
					if v.RoundMessage {
						// Телеграм-кружок смотрим до конца
						if v.Duration > 0 {
							return time.Duration(v.Duration) * time.Second
						}
						return randomDuration(2, 5)
					}
					// Обычное видео — 11–16 секунд
					return randomDuration(11, 16)
				}
			}
		}
		// Прочие документы считаем простым текстом
		return randomDuration(2, 5)
	default:
		// Текст, стикеры и прочее
		return randomDuration(2, 5)
	}
}

// randomDuration возвращает случайную продолжительность в заданном диапазоне секунд.
func randomDuration(min, max int) time.Duration {
	if max <= min {
		return time.Duration(min) * time.Second
	}
	return time.Duration(rand.Intn(max-min+1)+min) * time.Second
}
