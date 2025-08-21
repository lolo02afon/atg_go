package active_sessions_disconnect

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"atg_go/pkg/storage"
	"atg_go/pkg/telegram/module"

	"github.com/gotd/td/tg"
)

// DisconnectSuspiciousSessions отключает неактивные сессии для всех авторизованных аккаунтов.
// Перед проверкой каждого аккаунта добавляется случайная задержка,
// чтобы запросы в Telegram не выглядели подозрительно.
// Сессии отключаются, если они не текущие и их устройство не совпадает с разрешённым.
// minDelay и maxDelay задают границы задержки в секундах.
func DisconnectSuspiciousSessions(db *storage.DB, minDelay, maxDelay int) (map[string][]string, error) {
	accounts, err := db.GetAuthorizedAccounts()
	if err != nil {
		// Логируем ошибку, чтобы быстрее найти проблемы с БД
		log.Printf("[ACTIVE SESSIONS DISCONNECT] ошибка получения аккаунтов: %v", err)
		return nil, err
	}

	result := make(map[string][]string)

	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, acc := range accounts {
		// Выбираем задержку в заданном диапазоне, чтобы распределить нагрузку.
		if maxDelay > minDelay {
			delay := randSrc.Intn(maxDelay-minDelay+1) + minDelay
			time.Sleep(time.Duration(delay) * time.Second)
		}

		client, err := module.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID)
		if err != nil {
			// Не прерываем работу из-за одного аккаунта, чтобы обработать остальные
			log.Printf("[ACTIVE SESSIONS DISCONNECT] аккаунт %s: ошибка инициализации: %v", acc.Phone, err)
			continue
		}

		ctx := context.Background()
		if err := client.Run(ctx, func(ctx context.Context) error {
			api := tg.NewClient(client)
			auths, err := api.AccountGetAuthorizations(ctx)
			if err != nil {
				return err
			}
			for _, a := range auths.Authorizations {
				if a.Current {
					continue
				}
				if a.DeviceModel == "HP Laptop 14s-dq2xxx" {
					continue
				}
				if _, err := api.AccountResetAuthorization(ctx, a.Hash); err != nil {
					// Если отключение не удалось, то сессия продолжит работать.
					// Фиксируем проблему в таблице Sos, чтобы вовремя заметить сбой.
					msg := fmt.Sprintf("аккаунт %d (%s), устройство %s: %v", acc.ID, acc.Phone, a.DeviceModel, err)
					if saveErr := db.SaveSos(msg); saveErr != nil {
						log.Printf("[ACTIVE SESSIONS DISCONNECT] ошибка записи в Sos: %v", saveErr)
					}
					log.Printf("[ACTIVE SESSIONS DISCONNECT] аккаунт %s: не удалось отключить %s: %v", acc.Phone, a.DeviceModel, err)
					continue
				}
				log.Printf("[ACTIVE SESSIONS DISCONNECT] аккаунт %s: отключено устройство %s", acc.Phone, a.DeviceModel)
				result[acc.Phone] = append(result[acc.Phone], a.DeviceModel)
			}
			return nil
		}); err != nil {
			log.Printf("[ACTIVE SESSIONS DISCONNECT] аккаунт %s: ошибка обработки: %v", acc.Phone, err)
		}
	}

	return result, nil
}
