package active_sessions_disconnect

import (
	"context"
	"log"
	"math/rand"
	"time"

	"atg_go/pkg/storage"
	"atg_go/pkg/telegram/module"

	"github.com/gotd/td/tg"
)

// LogAuthorizations выводит в журнал данные активных сессий
// случайного авторизованного аккаунта. Это позволяет не захламлять
// логи данными всех аккаунтов одновременно и сразу видеть телефон
// выбранного аккаунта.
func LogAuthorizations(db *storage.DB) error {
	accounts, err := db.GetAuthorizedAccounts()
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		return nil
	}

	// Выбираем один аккаунт случайным образом, чтобы контролировать объём логов
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	acc := accounts[rnd.Intn(len(accounts))]

	// Инициализируем клиента Telegram для выбранного аккаунта
	client, err := module.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID)
	if err != nil {
		log.Printf("[ACTIVE SESSIONS] аккаунт %d (%s): ошибка инициализации: %v", acc.ID, acc.Phone, err)
		return nil
	}

	ctx := context.Background()
	if err := client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)
		auths, err := api.AccountGetAuthorizations(ctx)
		if err != nil {
			return err
		}
		// Логируем каждую сессию отдельно, добавляя телефон для наглядности
		for _, a := range auths.Authorizations {
			log.Printf("[ACTIVE SESSIONS] аккаунт %d (%s): %+v", acc.ID, acc.Phone, a)
		}
		return nil
	}); err != nil {
		log.Printf("[ACTIVE SESSIONS] аккаунт %d (%s): ошибка получения сессий: %v", acc.ID, acc.Phone, err)
	}
	return nil
}
