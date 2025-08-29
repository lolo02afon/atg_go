package channel_duplicate

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"atg_go/pkg/storage"
	base "atg_go/pkg/telegram/module"

	"github.com/gotd/td/tg"
)

// channelInfo хранит данные для пересылки сообщений.
type channelInfo struct {
	id     int
	donor  *tg.Channel
	target *tg.Channel
}

// Connect присоединяет модуль дублирования каналов к существующему клиенту Telegram.
// Модуль использует уже готовые api и диспетчер, не открывая сессию повторно.
func Connect(ctx context.Context, api *tg.Client, dispatcher *tg.UpdateDispatcher, db *storage.DB, accountID int) {
	dups, err := db.GetChannelDuplicates()
	if err != nil {
		log.Printf("[CHANNEL DUPLICATE] получение списка дубликатов: %v", err)
		return
	}
	if len(dups) == 0 {
		log.Printf("[CHANNEL DUPLICATE] нет каналов для дублирования")
		return
	}

	chMap := make(map[int64]channelInfo)

	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, upd *tg.UpdateNewChannelMessage) error {
		msg, ok := upd.Message.(*tg.Message)
		if !ok {
			return nil
		}
		peer, ok := msg.PeerID.(*tg.PeerChannel)
		if !ok {
			return nil
		}
		info, ok := chMap[peer.ChannelID]
		if !ok {
			return nil
		}
		updated, err := db.TrySetLastPostID(info.id, msg.ID)
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] обновление last_post_id: %v", err)
			return nil
		}
		if !updated {
			return nil
		}
		for i := 0; i < 3; i++ {
			_, err = api.MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
				FromPeer: &tg.InputPeerChannel{ChannelID: info.donor.ID, AccessHash: info.donor.AccessHash},
				ID:       []int{msg.ID},
				ToPeer:   &tg.InputPeerChannel{ChannelID: info.target.ID, AccessHash: info.target.AccessHash},
				RandomID: []int64{rand.Int63()},
			})
			if err == nil {
				return nil
			}
			if i == 2 {
				saveErr := db.SaveSos(fmt.Sprintf("не удалось переслать пост %d с канала %d: %v", msg.ID, info.donor.ID, err))
				if saveErr != nil {
					log.Printf("[CHANNEL DUPLICATE] ошибка записи в Sos: %v", saveErr)
				}
			}
			time.Sleep(2 * time.Second)
		}
		return nil
	})

	for _, cd := range dups {
		donorUser, err := base.Modf_ExtractUsername(cd.URLChannelDonor)
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] некорректная ссылка %s: %v", cd.URLChannelDonor, err)
			continue
		}
		donorResolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: donorUser})
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] не удалось получить канал %s: %v", cd.URLChannelDonor, err)
			continue
		}
		donorCh, err := base.Modf_FindChannel(donorResolved.GetChats())
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] канал %s не найден: %v", cd.URLChannelDonor, err)
			continue
		}
		if err := base.Modf_JoinChannel(ctx, api, donorCh, db, accountID); err != nil && !strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
			log.Printf("[CHANNEL DUPLICATE] подписка на %s: %v", cd.URLChannelDonor, err)
			continue
		}
		settings := tg.InputPeerNotifySettings{}
		settings.SetMuteUntil(0)
		_, err = api.AccountUpdateNotifySettings(ctx, &tg.AccountUpdateNotifySettingsRequest{
			Peer:     &tg.InputNotifyPeer{Peer: &tg.InputPeerChannel{ChannelID: donorCh.ID, AccessHash: donorCh.AccessHash}},
			Settings: settings,
		})
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] уведомления %s: %v", cd.URLChannelDonor, err)
		}
		if cd.ChannelDonorTGID == nil {
			_ = db.SetChannelDonorTGID(cd.ID, fmt.Sprintf("%d", donorCh.ID))
		}

		targetUser, err := base.Modf_ExtractUsername(cd.OrderURL)
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] некорректная ссылка %s: %v", cd.OrderURL, err)
			continue
		}
		targetResolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: targetUser})
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] не удалось получить канал %s: %v", cd.OrderURL, err)
			continue
		}
		targetCh, err := base.Modf_FindChannel(targetResolved.GetChats())
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] канал %s не найден: %v", cd.OrderURL, err)
			continue
		}
		if err := base.Modf_JoinChannel(ctx, api, targetCh, db, accountID); err != nil && !strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
			log.Printf("[CHANNEL DUPLICATE] подписка на %s: %v", cd.OrderURL, err)
		}
		if cd.OrderChannelTGID == nil {
			_ = db.SetOrderChannelTGID(cd.OrderID, fmt.Sprintf("%d", targetCh.ID))
		}
		chMap[donorCh.ID] = channelInfo{id: cd.ID, donor: donorCh, target: targetCh}
	}
}
