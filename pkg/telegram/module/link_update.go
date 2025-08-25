package module

import (
	"context"
	"fmt"
	"log"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	accountmutex "atg_go/pkg/telegram/module/account_mutex"

	"github.com/gotd/td/tg"
)

// Modf_OrderLinkUpdate обновляет описание у всех аккаунтов согласно их order_id
// Если order_id есть, в описание ставится текст из поля url_description соответствующего заказа,
// иначе описание очищается. Комментарии на русском языке по требованию пользователя.
func Modf_OrderLinkUpdate(db *storage.DB) error {
	// Сначала освобождаем аккаунты под мониторингом, если они случайно привязаны к заказам
	if err := db.ReleaseMonitoringAccounts(); err != nil {
		log.Printf("[LINK_UPDATE ERROR] освобождение мониторинговых аккаунтов: %v", err)
		return err
	}

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

	// Перед обновлением описаний заполняем отсутствующие ID каналов в заказах
	if err := populateOrderChannelTGID(db, accounts); err != nil {
		log.Printf("[LINK_UPDATE ERROR] заполнение channel_tgid: %v", err)
		return err
	}

	for _, acc := range accounts {
		var description string
		if acc.OrderID != nil {
			// Получаем текст для описания из заказа (поле url_description)
			order, err := db.GetOrderByID(*acc.OrderID)
			if err != nil {
				log.Printf("[LINK_UPDATE ERROR] заказ %d: %v", *acc.OrderID, err)
				continue
			}
			description = order.URLDescription
		}
		if err := updateAccountDescription(db, acc, description); err != nil {
			log.Printf("[LINK_UPDATE ERROR] аккаунт %d: %v", acc.ID, err)
		}
	}
	return nil
}

// populateOrderChannelTGID заполняет поле channel_tgid у заказов с помощью Telegram API
// Используется первый доступный авторизованный аккаунт, так как требуется только получение ID канала
func populateOrderChannelTGID(db *storage.DB, accounts []models.Account) error {
	orders, err := db.GetOrdersWithoutChannelTGID()
	if err != nil {
		return err
	}
	if len(orders) == 0 {
		return nil
	}
	if len(accounts) == 0 {
		return fmt.Errorf("нет авторизованных аккаунтов")
	}
	acc := accounts[0]

	// Инициализируем клиента Telegram для первого аккаунта
	client, err := Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	return client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)
		for _, o := range orders {
			username, err := Modf_ExtractUsername(o.URLDefault)
			if err != nil {
				log.Printf("[LINK_UPDATE WARN] неверный URL %s: %v", o.URLDefault, err)
				continue
			}
			resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
			if err != nil {
				log.Printf("[LINK_UPDATE WARN] resolve %s: %v", username, err)
				continue
			}
			ch, err := Modf_FindChannel(resolved.GetChats())
			if err != nil {
				log.Printf("[LINK_UPDATE WARN] поиск канала %s: %v", username, err)
				continue
			}
			if err := db.SetOrderChannelTGID(o.ID, ch.ID); err != nil {
				log.Printf("[LINK_UPDATE WARN] обновление channel_tgid заказа %d: %v", o.ID, err)
				continue
			}
		}
		return nil
	})
}

// updateAccountDescription устанавливает новое описание (about) для аккаунта
// Описание берётся из поля url_description заказа
func updateAccountDescription(db *storage.DB, acc models.Account, description string) error {
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

		// Если требуется установить описание, делаем второй запрос
		if description != "" {
			reqSet := tg.AccountUpdateProfileRequest{}
			reqSet.SetAbout(description)
			if _, err := api.AccountUpdateProfile(ctx, &reqSet); err != nil {
				return err
			}
		}
		return nil
	})
}
