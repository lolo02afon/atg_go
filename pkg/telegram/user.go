package telegram

import (
	"context"
	"fmt"
	"log"
	"time"

	module "atg_go/pkg/telegram/module"

	"github.com/gotd/td/tg"
)

// GetUserID возвращает Telegram user ID для указанного аккаунта (по номеру телефона и ключам API).
func GetUserID(phone string, apiID int, apiHash string) (int64, error) {
	// Инициализируем клиента через наш модуль, передавая apiID, apiHash и номер телефона
	client, err := module.Modf_AccountInitialization(apiID, apiHash, phone)
	if err != nil {
		// Если не удалось инициализировать клиента, возвращаем ошибку
		return 0, err
	}

	// Создаём контекст с таймаутом 30 секунд, чтобы запрос не висел бесконечно
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// Отложено отменяем контекст по выходу из функции
	defer cancel()

	var id int64
	// Запускаем сессию клиента и выполняем запрос внутри callback-функции
	err = client.Run(ctx, func(ctx context.Context) error {
		// Создаём обёртку API для удобного вызова методов
		api := tg.NewClient(client)

		// Отправляем запрос UsersGetFullUser, чтобы получить подробную информацию о текущем пользователе
		meFull, err := api.UsersGetFullUser(ctx, &tg.InputUserSelf{})
		if err != nil {
			// В случае ошибки запроса возвращаем её дальше
			return err
		}

		// Из полученного ответа извлекаем ID пользователя
		id = meFull.FullUser.ID
		return nil
	})
	if err != nil {
		// Если внутри Run произошла ошибка, оборачиваем её в более понятное сообщение
		return 0, fmt.Errorf("failed to get self id: %w", err)
	}

	log.Printf("(GetUserID) ID аккаунта: %s", id)

	// Возвращаем найденный ID и nil в качестве ошибки
	return id, nil
}
