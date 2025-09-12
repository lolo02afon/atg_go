package accounts_sessions_disconnect

import (
	"context"
	"log"

	"atg_go/pkg/storage"
	tech "atg_go/pkg/telegram/technical"

	"github.com/gotd/td/tg"
)

// CheckAccountsState проверяет доступность всех авторизованных аккаунтов
// через метод updates.getState. Возвращает список телефонов,
// к которым программа потеряла доступ.
func CheckAccountsState(db *storage.DB) ([]string, error) {
	accounts, err := db.GetAllAuthorizedAccounts()
	if err != nil {
		return nil, err
	}

	var lost []string
	for _, acc := range accounts {
		client, err := tech.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID, nil)
		if err != nil {
			log.Printf("[ACCOUNTS SESSIONS] аккаунт %d (%s): ошибка инициализации: %v", acc.ID, acc.Phone, err)
			lost = append(lost, acc.Phone)
			continue
		}

		ctx := context.Background()
		err = client.Run(ctx, func(ctx context.Context) error {
			api := tg.NewClient(client)
			_, err := api.UpdatesGetState(ctx)
			return err
		})
		if err != nil {
			log.Printf("[ACCOUNTS SESSIONS] аккаунт %d (%s): потерян доступ: %v", acc.ID, acc.Phone, err)
			lost = append(lost, acc.Phone)
			continue
		}
		// Фиксируем успешно проверенный аккаунт.
		if incErr := db.IncreaseAccountsCheck(); incErr != nil {
			log.Printf("[ACCOUNTS SESSIONS] ошибка увеличения счётчика: %v", incErr)
		}
	}

	return lost, nil
}
