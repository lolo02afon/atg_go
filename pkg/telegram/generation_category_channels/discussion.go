package generation_category_channels

import (
	"context"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	tech "atg_go/pkg/telegram/a_technical"

	"github.com/gotd/td/tg"
)

// HasAccessibleDiscussion проверяет, открыт ли чат обсуждения у канала
// и доступен ли он без дополнительных условий.
func HasAccessibleDiscussion(db *storage.DB, acc models.Account, channelURL string) (bool, error) {
	client, err := tech.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID, nil)
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var ok bool
	err = client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)

		username, err := tech.Modf_ExtractUsername(channelURL)
		if err != nil {
			return err
		}
		resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
		if err != nil {
			return err
		}
		ch, err := tech.Modf_FindChannel(resolved.GetChats())
		if err != nil {
			return err
		}

		full, err := api.ChannelsGetFullChannel(ctx, &tg.InputChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash})
		if err != nil {
			return err
		}
		fullChat, okFull := full.GetFullChat().(*tg.ChannelFull)
		if !okFull || fullChat.LinkedChatID == 0 {
			return nil
		}

		var discussion *tg.Channel
		for _, raw := range full.GetChats() {
			if c, ok := raw.(*tg.Channel); ok && c.ID == fullChat.LinkedChatID {
				discussion = c
				break
			}
		}
		if discussion == nil {
			return nil
		}

		// Проверяем, что чат обсуждения доступен сразу, без вступления
		if _, err := api.ChannelsGetChannels(ctx, []tg.InputChannelClass{&tg.InputChannel{ChannelID: discussion.ID, AccessHash: discussion.AccessHash}}); err != nil {
			return nil
		}

		ok = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return ok, nil
}
