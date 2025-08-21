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
// нескольких случайных (до пяти) авторизованных аккаунтов. Такой
// подход ограничивает объём логов и всё же даёт понимание о сессиях
// разных пользователей.
func LogAuthorizations(db *storage.DB) error {
	accounts, err := db.GetAuthorizedAccounts()
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		return nil
	}

	// Перемешиваем список, чтобы выбранные аккаунты были случайными
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	rnd.Shuffle(len(accounts), func(i, j int) { accounts[i], accounts[j] = accounts[j], accounts[i] })

	// Ограничиваем количество проверяемых аккаунтов пятью, чтобы не засорять логи
	limit := 5
	if len(accounts) < limit {
		limit = len(accounts)
	}

	// Для каждого выбранного аккаунта инициализируем клиента и выводим его активные сессии
	for _, acc := range accounts[:limit] {
		client, err := module.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID)
		if err != nil {
			// Не прерываем обработку остальных аккаунтов, чтобы увидеть информацию хотя бы по части из них
			log.Printf("[ACTIVE SESSIONS] аккаунт %d (%s): ошибка инициализации: %v", acc.ID, acc.Phone, err)
			continue
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
	}
	return nil
}
