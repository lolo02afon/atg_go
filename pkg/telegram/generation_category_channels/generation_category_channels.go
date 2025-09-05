package generation_category_channels

import (
	"context"
	"fmt"
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
		return nil, err
	}
	defer accountmutex.UnlockAccount(acc.ID)

	client, err := module.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID, nil)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var result []string
	err = client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)
		username, err := module.Modf_ExtractUsername(channelURL)
		if err != nil {
			return err
		}
		resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
		if err != nil {
			return err
		}
		ch, err := module.Modf_FindChannel(resolved.GetChats())
		if err != nil {
			return err
		}
		resp, err := api.ChannelsGetChannelRecommendations(ctx, &tg.ChannelsGetChannelRecommendationsRequest{
			Channel: &tg.InputChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash},
		})
		if err != nil {
			return err
		}
		for _, chat := range resp.GetChats() {
			if rec, ok := chat.(*tg.Channel); ok {
				if rec.Username != "" {
					result = append(result, "https://t.me/"+rec.Username)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("не удалось получить рекомендации: %w", err)
	}
	return result, nil
}
