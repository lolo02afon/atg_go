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
	id     int         // ID записи channel_duplicate
	donor  *tg.Channel // Канал-источник
	target *tg.Channel // Наш канал
	remove *string     // Текст для удаления из поста
	add    *string     // Текст для добавления
}

// getMessageID извлекает идентификатор пересланного сообщения из ответа Telegram.
func getMessageID(upd tg.UpdatesClass) (int, error) {
	switch u := upd.(type) {
	case *tg.Updates:
		for _, up := range u.Updates {
			if id, err := getMessageIDFromUpdate(up); err == nil {
				return id, nil
			}
		}
	case *tg.UpdatesCombined:
		for _, up := range u.Updates {
			if id, err := getMessageIDFromUpdate(up); err == nil {
				return id, nil
			}
		}
	case *tg.UpdateShort:
		return getMessageIDFromUpdate(u.Update)
	case *tg.UpdateShortMessage:
		return u.ID, nil
	case *tg.UpdateShortChatMessage:
		return u.ID, nil
	case *tg.UpdateShortSentMessage:
		return u.ID, nil
	}
	return 0, fmt.Errorf("не удалось получить ID пересланного сообщения")
}

// getMessageIDFromUpdate разбирает отдельное обновление и ищет ID сообщения.
func getMessageIDFromUpdate(up tg.UpdateClass) (int, error) {
	switch v := up.(type) {
	case *tg.UpdateNewMessage:
		if m, ok := v.Message.(*tg.Message); ok {
			return m.ID, nil
		}
	case *tg.UpdateNewChannelMessage:
		if m, ok := v.Message.(*tg.Message); ok {
			return m.ID, nil
		}
	case *tg.UpdateNewScheduledMessage:
		if m, ok := v.Message.(*tg.Message); ok {
			return m.ID, nil
		}
	}
	return 0, fmt.Errorf("ID не найден")
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
		// Перед пересылкой проверяем пост на признаки рекламы
		if isAdvertisement(msg) {
			return nil
		}
		// Планируем отправку поста через четыре месяца без указания источника
		schedule := int(time.Now().AddDate(0, 4, 0).Unix())
		req := &tg.MessagesForwardMessagesRequest{
			FromPeer:   &tg.InputPeerChannel{ChannelID: info.donor.ID, AccessHash: info.donor.AccessHash},
			ID:         []int{msg.ID},
			ToPeer:     &tg.InputPeerChannel{ChannelID: info.target.ID, AccessHash: info.target.AccessHash},
			RandomID:   []int64{rand.Int63()},
			DropAuthor: true,
		}
		req.SetScheduleDate(schedule)

		res, err := api.MessagesForwardMessages(ctx, req)
		if err != nil {
			saveErr := db.SaveSos(fmt.Sprintf("не удалось переслать пост %d с канала %d: %v", msg.ID, info.donor.ID, err))
			if saveErr != nil {
				log.Printf("[CHANNEL DUPLICATE] ошибка записи в Sos: %v", saveErr)
			}
			return nil
		}

		// Извлекаем ID отложенного сообщения
		forwardedID, err := getMessageID(res)
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] получение ID сообщения: %v", err)
			return nil
		}

		// Подготавливаем текст: удаляем указанную фразу и добавляем суффикс
		text := msg.Message
		if info.remove != nil && *info.remove != "" {
			if strings.Contains(text, *info.remove) {
				text = strings.ReplaceAll(text, *info.remove, "")
			} else {
				// Если ничего не удалено, оставляем пост в отложенных сообщениях
				return nil
			}
		}
		if info.add != nil {
			text += *info.add
		}

		// Редактируем отложенный пост
		editReq := tg.MessagesEditMessageRequest{
			Peer: &tg.InputPeerChannel{ChannelID: info.target.ID, AccessHash: info.target.AccessHash},
			ID:   forwardedID,
		}
		editReq.SetMessage(text)
		_, err = api.MessagesEditMessage(ctx, &editReq)
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] редактирование сообщения: %v", err)
			return nil
		}

		// Публикуем сообщение немедленно
		_, err = api.MessagesSendScheduledMessages(ctx, &tg.MessagesSendScheduledMessagesRequest{
			Peer: &tg.InputPeerChannel{ChannelID: info.target.ID, AccessHash: info.target.AccessHash},
			ID:   []int{forwardedID},
		})
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] публикация отложенного сообщения: %v", err)
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
		chMap[donorCh.ID] = channelInfo{id: cd.ID, donor: donorCh, target: targetCh, remove: cd.PostTextRemove, add: cd.PostTextAdd}
	}
}
