package accounts_auth

import (
	"context"
	"fmt"
	"log"

	"atg_go/models"
	"atg_go/pkg/storage"
	"atg_go/pkg/telegram/technical"

	"github.com/gotd/td/session"
	"github.com/gotd/td/tg"
)

// Check проверяет, сохранилась ли авторизация аккаунта.
// При отсутствии сессии или ошибке Telegram фиксируем событие в Sos
// и сбрасываем флаг авторизации.
func Check(db *storage.DB, acc models.Account) bool {
	client, err := module.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID, nil)
	if err != nil {
		// Инициализация клиента без сессии невозможна, считаем аккаунт неавторизованным.
		log.Printf("[ACCOUNT AUTH CHECK] аккаунт %s: ошибка инициализации: %v", acc.Phone, err)
		markUnauthorized(db, acc)
		return false
	}

	ctx := context.Background()
	err = client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)
		_, err := api.UsersGetFullUser(ctx, &tg.InputUserSelf{})
		return err
	})
	if err != nil {
		// Любая ошибка при запросе означает, что сессия недействительна или отсутствует.
		if err == session.ErrNotFound {
			log.Printf("[ACCOUNT AUTH CHECK] аккаунт %s: сессия не найдена", acc.Phone)
		} else {
			log.Printf("[ACCOUNT AUTH CHECK] аккаунт %s: ошибка запроса: %v", acc.Phone, err)
		}
		markUnauthorized(db, acc)
		return false
	}

	// Авторизация присутствует, дополнительных действий не требуется.
	return true
}

// markUnauthorized сбрасывает флаг авторизации и пишет сообщение в Sos.
func markUnauthorized(db *storage.DB, acc models.Account) {
	if err := db.MarkAccountAsUnauthorized(acc.ID); err != nil {
		log.Printf("[ACCOUNT AUTH CHECK] ошибка обновления статуса %s: %v", acc.Phone, err)
	}
	msg := fmt.Sprintf("номер %s больше не авторизован в программе", acc.Phone)
	if err := db.SaveSos(msg); err != nil {
		log.Printf("[ACCOUNT AUTH CHECK] ошибка записи в Sos: %v", err)
	}
}
