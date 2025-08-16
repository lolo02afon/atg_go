package module

import (
	"context"
	"log"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	accountmutex "atg_go/pkg/telegram/module/account_mutex"

	"github.com/gotd/td/tg"
)

// Modf_OrderLinkUpdate обновляет описание у всех аккаунтов согласно их order_id
// Если order_id есть, в описание ставится ссылка из соответствующего заказа,
// иначе описание очищается. Комментарии на русском языке по требованию пользователя.
func Modf_OrderLinkUpdate(db *storage.DB) error {
	// Перед обновлением описаний синхронизируем количество аккаунтов в заказах
	if err := db.AssignFreeAccountsToOrders(); err != nil {
		log.Printf("[LINK_UPDATE ERROR] назначение аккаунтов: %v", err)
		return err
	}

	// Получаем все авторизованные аккаунты
	accounts, err := db.GetAuthorizedAccounts()
	if err != nil {
		log.Printf("[LINK_UPDATE ERROR] выборка аккаунтов: %v", err)
		return err
	}

	for _, acc := range accounts {
		var link string
		if acc.OrderID != nil {
			// Получаем URL заказа для подстановки в описание
			order, err := db.GetOrderByID(*acc.OrderID)
			if err != nil {
				log.Printf("[LINK_UPDATE ERROR] заказ %d: %v", *acc.OrderID, err)
				continue
			}
			link = order.URL
		}
		if err := updateAccountLink(db, acc, link); err != nil {
			log.Printf("[LINK_UPDATE ERROR] аккаунт %d: %v", acc.ID, err)
		}
	}
	return nil
}

// updateAccountLink устанавливает новое описание (about) для аккаунта
func updateAccountLink(db *storage.DB, acc models.Account, link string) error {
	// Блокируем аккаунт, чтобы не выполнять параллельные операции
	if err := accountmutex.LockAccount(acc.ID); err != nil {
		return err
	}
	defer accountmutex.UnlockAccount(acc.ID)

	// Инициализируем клиента Telegram
	client, err := Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)
		// Сначала очищаем текущее описание, чтобы гарантированно заменить его
		reqClear := tg.AccountUpdateProfileRequest{}
		reqClear.SetAbout("")
		if _, err := api.AccountUpdateProfile(ctx, &reqClear); err != nil {
			return err
		}

		// Если требуется поставить ссылку, делаем второй запрос
		if link != "" {
			reqSet := tg.AccountUpdateProfileRequest{}
			reqSet.SetAbout(link)
			if _, err := api.AccountUpdateProfile(ctx, &reqSet); err != nil {
				return err
			}
		}
		return nil
	})
}
