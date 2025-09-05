package generation_category_channels

import (
	"context"
	"fmt"
	"log"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	module "atg_go/pkg/telegram/module"
	accountmutex "atg_go/pkg/telegram/module/account_mutex"

	"github.com/gotd/td/tg"
)

// GetChannelRecommendations возвращает список похожих каналов для указанного канала.
// Работает от имени заданного аккаунта.
func GetChannelRecommendations(db *storage.DB, acc models.Account, channelURL string) ([]string, error) {
	if err := accountmutex.LockAccount(acc.ID); err != nil {
		// Фиксируем проблемы при блокировке аккаунта
		log.Printf("[GENERATION ERROR] не удалось заблокировать аккаунт %d: %v", acc.ID, err)
		return nil, err
	}
	defer accountmutex.UnlockAccount(acc.ID)

	client, err := module.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID, nil)
	if err != nil {
		// Логируем ошибку инициализации клиента
		log.Printf("[GENERATION ERROR] не удалось инициализировать аккаунт %d: %v", acc.ID, err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var result []string
	err = client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)
		username, err := module.Modf_ExtractUsername(channelURL)
		if err != nil {
			// Ошибка извлечения имени пользователя из ссылки
			log.Printf("[GENERATION ERROR] не удалось извлечь имя пользователя из %s: %v", channelURL, err)
			return err
		}
		resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
		if err != nil {
			// Ошибка запроса к API Telegram
			log.Printf("[GENERATION ERROR] не удалось разрешить имя %s: %v", username, err)
			return err
		}
		ch, err := module.Modf_FindChannel(resolved.GetChats())
		if err != nil {
			// Канал не найден в ответе Telegram
			log.Printf("[GENERATION ERROR] не найден канал %s: %v", username, err)
			return err
		}
		resp, err := api.ChannelsGetChannelRecommendations(ctx, &tg.ChannelsGetChannelRecommendationsRequest{
			Channel: &tg.InputChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash},
		})
		if err != nil {
			// Ошибка получения рекомендаций по каналу
			log.Printf("[GENERATION ERROR] не удалось получить рекомендации для %s: %v", username, err)
			return err
		}
		for _, chat := range resp.GetChats() {
			if rec, ok := chat.(*tg.Channel); ok {
				if rec.Username != "" {
					result = append(result, "https://t.me/"+rec.Username)
					// Фиксируем каждый десятый найденный канал
					if len(result)%10 == 0 {
						log.Printf("[GENERATION INFO] получено %d рекомендаций, последний: %s", len(result), rec.Username)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		// Итоговая ошибка выполнения
		log.Printf("[GENERATION ERROR] не удалось получить рекомендации: %v", err)
		return nil, fmt.Errorf("не удалось получить рекомендации: %w", err)
	}
	return result, nil
}
