package active_sessions_disconnect

import (
	"context"
	"log"

	"atg_go/pkg/storage"
	"atg_go/pkg/telegram/module"

	"github.com/gotd/td/tg"
)

// LogAuthorizations выводит в журнал данные всех активных сессий
// для каждого авторизованного аккаунта. Это помогает оперативно
// проверить, какие устройства используют аккаунты в данный момент.
func LogAuthorizations(db *storage.DB) error {
	accounts, err := db.GetAuthorizedAccounts()
	if err != nil {
		return err
	}

	for _, acc := range accounts {
		// Инициализируем клиента Telegram для аккаунта
		client, err := module.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID)
		if err != nil {
			log.Printf("[ACTIVE SESSIONS] аккаунт %d: ошибка инициализации: %v", acc.ID, err)
			continue
		}

		ctx := context.Background()
		if err := client.Run(ctx, func(ctx context.Context) error {
			api := tg.NewClient(client)
			auths, err := api.AccountGetAuthorizations(ctx)
			if err != nil {
				return err
			}
			// Логируем каждую сессию отдельно, чтобы сохранить максимум деталей
			for _, a := range auths.Authorizations {
				log.Printf("[ACTIVE SESSIONS] аккаунт %d: %+v", acc.ID, a)
			}
			return nil
		}); err != nil {
			log.Printf("[ACTIVE SESSIONS] аккаунт %d: ошибка получения сессий: %v", acc.ID, err)
		}
	}
	return nil
}
