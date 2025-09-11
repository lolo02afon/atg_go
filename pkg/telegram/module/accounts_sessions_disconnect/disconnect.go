package accounts_sessions_disconnect

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"atg_go/pkg/storage"
	"atg_go/pkg/telegram/module"
	accountauthcheck "atg_go/pkg/telegram/module/account_auth_check"

	"github.com/gotd/td/tg"
)

// allowedDeviceModels содержит модели устройств, с которых сессии не отключаются.
var allowedDeviceModels = map[string]struct{}{
	"HP Laptop 14s-dq2xxx": {},
	"GL63 8RE":             {},
}

// DisconnectSuspiciousSessions отключает неактивные сессии для всех авторизованных аккаунтов.
// Перед проверкой каждого аккаунта добавляется случайная задержка,
// чтобы запросы в Telegram не выглядели подозрительно.
// Сессии отключаются, если они не текущие и их устройство не входит в список разрешённых.
// minDelay и maxDelay задают границы задержки в секундах.
func DisconnectSuspiciousSessions(db *storage.DB, minDelay, maxDelay int) (map[string][]string, error) {
	accounts, err := db.GetAuthorizedAccounts()
	if err != nil {
		// Логируем ошибку, чтобы быстрее найти проблемы с БД
		log.Printf("[ACCOUNTS SESSIONS DISCONNECT] ошибка получения аккаунтов: %v", err)
		return nil, err
	}

	result := make(map[string][]string)

	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, acc := range accounts {
		// Перед проверкой сессий убеждаемся, что аккаунт ещё авторизован.
		// Если авторизация пропала, пропускаем обработку, чтобы не тратить ресурсы впустую.
		if !accountauthcheck.Check(db, acc) {
			continue
		}

		// Выбираем задержку в заданном диапазоне, чтобы распределить нагрузку.
		if maxDelay > minDelay {
			delay := randSrc.Intn(maxDelay-minDelay+1) + minDelay
			time.Sleep(time.Duration(delay) * time.Second)
		}

		client, err := module.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID, nil)
		if err != nil {
			// Не прерываем работу из-за одного аккаунта, чтобы обработать остальные
			log.Printf("[ACCOUNTS SESSIONS DISCONNECT] аккаунт %s: ошибка инициализации: %v", acc.Phone, err)
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
				if _, allowed := allowedDeviceModels[a.DeviceModel]; allowed {
					// Пропускаем сессию, если устройство разрешено
					continue
				}
				if _, err := api.AccountResetAuthorization(ctx, a.Hash); err != nil {
					// Если отключение не удалось, то сессия продолжит работать.
					// Фиксируем проблему в таблице Sos, чтобы вовремя заметить сбой.
					msg := fmt.Sprintf("аккаунт %d (%s), устройство %s: %v", acc.ID, acc.Phone, a.DeviceModel, err)
					if saveErr := db.SaveSos(msg); saveErr != nil {
						log.Printf("[ACCOUNTS SESSIONS DISCONNECT] ошибка записи в Sos: %v", saveErr)
					}
					log.Printf("[ACCOUNTS SESSIONS DISCONNECT] аккаунт %s: не удалось отключить %s: %v", acc.Phone, a.DeviceModel, err)
					continue
				}
				log.Printf("[ACCOUNTS SESSIONS DISCONNECT] аккаунт %s: отключено устройство %s", acc.Phone, a.DeviceModel)
				result[acc.Phone] = append(result[acc.Phone], a.DeviceModel)
				// Увеличиваем счётчик отключённых сессий.
				if incErr := db.IncreaseAccountsSessionsDisconnect(); incErr != nil {
					log.Printf("[ACCOUNTS SESSIONS DISCONNECT] ошибка увеличения счётчика сессий: %v", incErr)
				}
			}
			return nil
		}); err != nil {
			log.Printf("[ACCOUNTS SESSIONS DISCONNECT] аккаунт %s: ошибка обработки: %v", acc.Phone, err)
			continue
		}
		// Фиксируем успешно проверенный аккаунт.
		if incErr := db.IncreaseAccountsCheck(); incErr != nil {
			log.Printf("[ACCOUNTS SESSIONS DISCONNECT] ошибка увеличения счётчика аккаунтов: %v", incErr)
		}
	}

	return result, nil
}
