package module

import (
	"context"
	"log"

	"atg_go/models"
	"atg_go/pkg/storage"

	"github.com/gotd/td/tg"
)

// ModF_UnsubscribeAll отключает все аккаунты от каналов и групп.
func ModF_UnsubscribeAll(db *storage.DB) error {
	accounts, err := db.GetAuthorizedAccounts()
	if err != nil {
		return err
	}
	for _, acc := range accounts {
		if err := unsubscribeAccount(db, &acc); err != nil {
			log.Printf("[UNSUBSCRIBE] аккаунт %d: %v", acc.ID, err)
		}
	}
	return nil
}

// unsubscribeAccount выходит из всех каналов и групп для одного аккаунта.
func unsubscribeAccount(db *storage.DB, acc *models.Account) error {
	client, err := Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)
		res, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{Limit: 100})
		if err != nil {
			return err
		}
		dialogs, ok := res.AsModified()
		if !ok {
			return nil
		}
		for _, raw := range dialogs.GetDialogs() {
			d, ok := raw.(*tg.Dialog)
			if !ok {
				continue
			}
			switch peer := d.Peer.(type) {
			case *tg.PeerChannel:
				if ch := findChannel(dialogs.GetChats(), peer.ChannelID); ch != nil {
					if _, err := api.ChannelsLeaveChannel(ctx, &tg.InputChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}); err != nil {
						log.Printf("[UNSUBSCRIBE] не удалось покинуть канал %d: %v", ch.ID, err)
					}
				}
			case *tg.PeerChat:
				if _, err := api.MessagesDeleteChatUser(ctx, &tg.MessagesDeleteChatUserRequest{ChatID: peer.ChatID, UserID: &tg.InputUserSelf{}}); err != nil {
					log.Printf("[UNSUBSCRIBE] не удалось покинуть группу %d: %v", peer.ChatID, err)
				}
			}
		}
		return nil
	})
}

// findChannel ищет канал по идентификатору среди чатов.
func findChannel(chats []tg.ChatClass, id int64) *tg.Channel {
	for _, ch := range chats {
		if c, ok := ch.(*tg.Channel); ok && c.ID == id {
			return c
		}
	}
	return nil
}
