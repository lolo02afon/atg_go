package channel_duplicate

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

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

// forwardScheduled пересылает сообщение в отложенные и возвращает его ID.
// Идентификатор извлекается из ответа Telegram по random_id.
func forwardScheduled(ctx context.Context, api *tg.Client, donor *tg.Channel, target *tg.Channel, msgID int, schedule int) (int, error) {
	randomID := rand.Int63()
	req := &tg.MessagesForwardMessagesRequest{
		FromPeer:   &tg.InputPeerChannel{ChannelID: donor.ID, AccessHash: donor.AccessHash},
		ID:         []int{msgID},
		ToPeer:     &tg.InputPeerChannel{ChannelID: target.ID, AccessHash: target.AccessHash},
		RandomID:   []int64{randomID},
		DropAuthor: true,
	}
	req.SetScheduleDate(schedule)

	upd, err := api.MessagesForwardMessages(ctx, req)
	if err != nil {
		return 0, err
	}

	var updates []tg.UpdateClass
	switch u := upd.(type) {
	case *tg.Updates:
		updates = u.Updates
	case *tg.UpdatesCombined:
		updates = u.Updates
	default:
		return 0, fmt.Errorf("неожиданный тип ответа %T", upd)
	}
	for _, u := range updates {
		switch up := u.(type) {
		case *tg.UpdateMessageID:
			if up.RandomID == randomID {
				return up.ID, nil
			}
		case *tg.UpdateNewScheduledMessage:
			if m, ok := up.Message.(*tg.Message); ok {
				return m.ID, nil
			}
		}
	}
	return 0, fmt.Errorf("ID отложенного сообщения не найден")
}

// parseTextURL ищет в тексте ссылку формата [текст](url)
// и возвращает сущность для Telegram и очищенный текст без служебных символов.
func parseTextURL(text string, baseOffset int) (*tg.MessageEntityTextURL, string) {
	markdown := regexp.MustCompile(`\[([^\]]+)\]\((https?://[^\s]+)\)`)
	if loc := markdown.FindStringSubmatchIndex(text); loc != nil {
		visible := text[loc[2]:loc[3]]
		url := text[loc[4]:loc[5]]
		offset := baseOffset + utf8.RuneCountInString(text[:loc[2]])
		length := utf8.RuneCountInString(visible)
		clean := text[:loc[0]] + visible + text[loc[1]:]
		ent := &tg.MessageEntityTextURL{Offset: offset, Length: length, URL: url}
		return ent, clean
	}
	return nil, text
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
		updated, remove, add, err := db.TrySetLastPostID(info.id, msg.ID)
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] обновление last_post_id: %v", err)
			return nil
		}
		if !updated {
			return nil
		}
		// Обновляем тексты в карте, чтобы использовать свежие значения
		info.remove = remove
		info.add = add
		chMap[peer.ChannelID] = info
		// Перед пересылкой проверяем пост на признаки рекламы
		if isAdvertisement(msg) {
			return nil
		}
		// Формируем текст для публикации
		text := msg.Message
		needsEdit := false
		if info.remove != nil && *info.remove != "" {
			if strings.Contains(text, *info.remove) {
				// Удаляем указанный фрагмент
				text = strings.ReplaceAll(text, *info.remove, "")
				needsEdit = true
			} else {
				// Если требуемый фрагмент не найден, оставляем пост в отложенных
				schedule := int(time.Now().AddDate(0, 4, 0).Unix())
				req := &tg.MessagesForwardMessagesRequest{
					FromPeer:   &tg.InputPeerChannel{ChannelID: info.donor.ID, AccessHash: info.donor.AccessHash},
					ID:         []int{msg.ID},
					ToPeer:     &tg.InputPeerChannel{ChannelID: info.target.ID, AccessHash: info.target.AccessHash},
					RandomID:   []int64{rand.Int63()},
					DropAuthor: true,
				}
				req.SetScheduleDate(schedule)
				if _, err := api.MessagesForwardMessages(ctx, req); err != nil {
					saveErr := db.SaveSos(fmt.Sprintf("не удалось переслать пост %d с канала %d: %v", msg.ID, info.donor.ID, err))
					if saveErr != nil {
						log.Printf("[CHANNEL DUPLICATE] ошибка записи в Sos: %v", saveErr)
					}
				}
				return nil
			}
		}
		// Сохраняем текст после удаления, чтобы знать смещение добавленного блока
		baseText := text
		var addText string
		if info.add != nil && *info.add != "" {
			// Добавляем текст в конец поста
			addText = *info.add
			text = baseText + addText
			needsEdit = true
		}

		// Если требуются изменения, работаем через отложенный пост
		if needsEdit {
			schedule := int(time.Now().AddDate(0, 4, 0).Unix())
			forwardedID, err := forwardScheduled(ctx, api, info.donor, info.target, msg.ID, schedule)
			if err != nil {
				saveErr := db.SaveSos(fmt.Sprintf("не удалось переслать пост %d с канала %d: %v", msg.ID, info.donor.ID, err))
				if saveErr != nil {
					log.Printf("[CHANNEL DUPLICATE] ошибка записи в Sos: %v", saveErr)
				}
				return nil
			}
			editReq := tg.MessagesEditMessageRequest{
				Peer: &tg.InputPeerChannel{ChannelID: info.target.ID, AccessHash: info.target.AccessHash},
				ID:   forwardedID,
			}
			editReq.SetMessage(text)
			// Анализируем добавленный текст на наличие ссылки и формируем сущность
			if addText != "" {
				if ent, clean := parseTextURL(addText, utf8.RuneCountInString(baseText)); ent != nil {
					editReq.SetMessage(baseText + clean)
					editReq.SetEntities([]tg.MessageEntityClass{ent})
				}
			}
			editReq.SetScheduleDate(schedule)
			if _, err = api.MessagesEditMessage(ctx, &editReq); err != nil {
				log.Printf("[CHANNEL DUPLICATE] редактирование сообщения: %v", err)
				return nil
			}
			if _, err = api.MessagesSendScheduledMessages(ctx, &tg.MessagesSendScheduledMessagesRequest{
				Peer: &tg.InputPeerChannel{ChannelID: info.target.ID, AccessHash: info.target.AccessHash},
				ID:   []int{forwardedID},
			}); err != nil {
				log.Printf("[CHANNEL DUPLICATE] публикация отложенного сообщения: %v", err)
			}
			return nil
		}

		// Изменения не требуются — пересылаем пост сразу
		if _, err := api.MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
			FromPeer:   &tg.InputPeerChannel{ChannelID: info.donor.ID, AccessHash: info.donor.AccessHash},
			ID:         []int{msg.ID},
			ToPeer:     &tg.InputPeerChannel{ChannelID: info.target.ID, AccessHash: info.target.AccessHash},
			RandomID:   []int64{rand.Int63()},
			DropAuthor: true,
		}); err != nil {
			saveErr := db.SaveSos(fmt.Sprintf("не удалось переслать пост %d с канала %d: %v", msg.ID, info.donor.ID, err))
			if saveErr != nil {
				log.Printf("[CHANNEL DUPLICATE] ошибка записи в Sos: %v", saveErr)
			}
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
