package telegram

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// Запрашиваем код у Телеграма для номера
func RequestCode(apiID int, apiHash, phone string) error {
	// Настройка генератора случайных чисел — нужно для клиентской работы
	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))

	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		// Используем in-memory storage — позже можно заменить на файл
		SessionStorage: &telegram.FileSessionStorage{Path: phone + ".session.json"},
		// Используется генератор
		Random: randSrc,
	})

	ctx := context.Background()

	return client.Run(ctx, func(ctx context.Context) error {
		// Используем готовый flow, чтобы запросить код
		flow := auth.NewFlow(
			auth.CodeOnly(phone, auth.CodeAuthenticatorFunc(func(ctx context.Context, sent *tg.AuthSentCode) (string, error) {
				// Flow только запрашивает код и возвращает ошибку не требуется здесь
				log.Printf("Код отправлен. Код хэш: %s", sent.PhoneCodeHash)
				return "", nil
			})),
			auth.SendCodeOptions{},
		)

		if err := flow.Run(ctx, client.Auth()); err != nil {
			log.Println("Ошибка запроса кода:", err)
			return err
		}

		log.Println("Запрос кода выполнен аккуратно")
		return nil
	})
}

func CompleteAuthorization(apiID int, apiHash, phone, code string) error {
	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))

	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{Path: phone + ".session.json"},
		Random:         randSrc,
	})

	ctx := context.Background()

	return client.Run(ctx, func(ctx context.Context) error {
		flow := auth.NewFlow(
			auth.Constant(phone, code, auth.CodeAuthenticatorFunc(func(ctx context.Context, _ *tg.AuthSentCode) (string, error) {
				return code, nil
			})),
			auth.SendCodeOptions{},
		)

		if err := flow.Run(ctx, client.Auth()); err != nil {
			return err
		}

		log.Println("Авторизация прошла успешно")
		return nil
	})
}
