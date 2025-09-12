package telegram

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"atg_go/pkg/storage"
	tgdup "atg_go/pkg/telegram/a_base/channel_duplicate"
	base "atg_go/pkg/telegram/a_technical"
	accountmutex "atg_go/pkg/telegram/a_technical/account_mutex"
	tgmonitor "atg_go/pkg/telegram/a_technical/monitoring"

	"github.com/gotd/td/tg"
)

// Run запускает все telegram-модули, используя одну сессию.
// Сессия создаётся один раз, после чего к ней подключаются модули.
func Run(db *storage.DB) {
	go func() {
		if err := run(db); err != nil {
			log.Printf("[TELEGRAM] остановлено: %v", err)
		}
	}()
}

// run инициализирует клиента и подключает модули.
func run(db *storage.DB) error {
	rand.Seed(time.Now().UnixNano())
	accounts, err := db.GetMonitoringAccounts()
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		return fmt.Errorf("нет аккаунтов для мониторинга")
	}
	acc := accounts[0]

	if err := accountmutex.LockAccount(acc.ID); err != nil {
		return err
	}
	defer accountmutex.UnlockAccount(acc.ID)

	dispatcher := tg.NewUpdateDispatcher()

	client, err := base.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID, dispatcher)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)
		// Передаём указатель на диспетчер, так как модули ожидают *tg.UpdateDispatcher.
		tgmonitor.Connect(ctx, api, &dispatcher, db, acc.ID)
		tgdup.Connect(ctx, api, &dispatcher, db, acc.ID)
		<-ctx.Done()
		return nil
	})
}
