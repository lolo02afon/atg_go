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

// copyMessage публикует копию сообщения в целевом канале.
// replyTo указывает, на какое сообщение нужно ответить в целевом канале.
func copyMessage(ctx context.Context, api *tg.Client, donor *tg.Channel, target *tg.Channel, msg *tg.Message, text string, entities []tg.MessageEntityClass, disablePreview bool, replyTo *int) error {
	peer := &tg.InputPeerChannel{ChannelID: target.ID, AccessHash: target.AccessHash}

	// Формируем ответ на указанное сообщение
	var reply tg.InputReplyToClass
	if replyTo != nil {
		reply = &tg.InputReplyToMessage{ReplyToMsgID: *replyTo}
	}

	switch m := msg.Media.(type) {
	case nil, *tg.MessageMediaEmpty, *tg.MessageMediaWebPage:
		// Текстовое сообщение без вложений
		req := &tg.MessagesSendMessageRequest{
			Peer:     peer,
			Message:  text,
			RandomID: rand.Int63(),
		}
		if reply != nil {
			req.SetReplyTo(reply)
		}
		if disablePreview {
			req.SetNoWebpage(true)
		}
		if len(entities) > 0 {
			req.SetEntities(entities)
		}
		_, err := api.MessagesSendMessage(ctx, req)
		return err
	case *tg.MessageMediaPhoto:
		// Копируем фото
		photo, ok := m.Photo.(*tg.Photo)
		if !ok {
			return fmt.Errorf("фото отсутствует в медиа")
		}
		media := &tg.InputMediaPhoto{ID: &tg.InputPhoto{ID: photo.ID, AccessHash: photo.AccessHash, FileReference: photo.FileReference}}
		req := &tg.MessagesSendMediaRequest{
			Peer:     peer,
			Media:    media,
			Message:  text,
			RandomID: rand.Int63(),
		}
		if reply != nil {
			req.SetReplyTo(reply)
		}
		if len(entities) > 0 {
			req.SetEntities(entities)
		}
		_, err := api.MessagesSendMedia(ctx, req)
		return err
	case *tg.MessageMediaDocument:
		// Копируем документ (видео, аудио, кружочки и т.п.)
		doc, ok := m.Document.(*tg.Document)
		if !ok {
			return fmt.Errorf("документ отсутствует в медиа")
		}
		media := &tg.InputMediaDocument{ID: &tg.InputDocument{ID: doc.ID, AccessHash: doc.AccessHash, FileReference: doc.FileReference}}
		req := &tg.MessagesSendMediaRequest{
			Peer:     peer,
			Media:    media,
			Message:  text,
			RandomID: rand.Int63(),
		}
		if reply != nil {
			req.SetReplyTo(reply)
		}
		if len(entities) > 0 {
			req.SetEntities(entities)
		}
		_, err := api.MessagesSendMedia(ctx, req)
		return err
	default:
		// Неизвестный тип медиа — пересылаем, чтобы не потерять сообщение
		fReq := &tg.MessagesForwardMessagesRequest{
			FromPeer:   &tg.InputPeerChannel{ChannelID: donor.ID, AccessHash: donor.AccessHash},
			ID:         []int{msg.ID},
			ToPeer:     peer,
			RandomID:   []int64{rand.Int63()},
			DropAuthor: true,
		}
		if reply != nil {
			fReq.SetReplyTo(reply)
		}
		_, err := api.MessagesForwardMessages(ctx, fReq)
		return err
	}
}

// findReplyTarget подбирает сообщение в целевом канале, к которому нужно привязать ответ.
// Если подходящее сообщение не найдено, возвращается 0.
func findReplyTarget(ctx context.Context, api *tg.Client, donor *tg.Channel, target *tg.Channel, msg *tg.Message) (int, error) {
	// Проверяем, является ли сообщение ответом в исходном канале
	replyHdr, ok := msg.GetReplyTo()
	if !ok {
		return 0, nil
	}
	hdr, ok := replyHdr.(*tg.MessageReplyHeader)
	if !ok {
		return 0, nil
	}
	parentID, ok := hdr.GetReplyToMsgID()
	if !ok {
		return 0, nil
	}

	// Получаем дату родительского сообщения на канале-источнике
	res, err := api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
		Channel: &tg.InputChannel{ChannelID: donor.ID, AccessHash: donor.AccessHash},
		ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: parentID}},
	})
	if err != nil {
		return 0, err
	}
	var parentDate int
	switch m := res.(type) {
	case *tg.MessagesChannelMessages:
		if len(m.Messages) == 0 {
			return 0, nil
		}
		if pm, ok := m.Messages[0].(*tg.Message); ok {
			parentDate = pm.Date
		} else {
			return 0, nil
		}
	case *tg.MessagesMessages:
		if len(m.Messages) == 0 {
			return 0, nil
		}
		if pm, ok := m.Messages[0].(*tg.Message); ok {
			parentDate = pm.Date
		} else {
			return 0, nil
		}
	default:
		return 0, fmt.Errorf("неожиданный тип ответа %T", res)
	}

	peer := &tg.InputPeerChannel{ChannelID: target.ID, AccessHash: target.AccessHash}
	// Сначала ищем сообщение с тем же временем
	searchReq := &tg.MessagesSearchRequest{
		Peer:    peer,
		Q:       "",
		Filter:  &tg.InputMessagesFilterEmpty{},
		MinDate: parentDate,
		MaxDate: parentDate,
		Limit:   1,
	}
	sr, err := api.MessagesSearch(ctx, searchReq)
	if err != nil {
		return 0, err
	}
	cand := extractFirstMessage(sr)
	if cand == nil {
		// Берём ближайший пост, опубликованный после указанного времени
		searchReq.MaxDate = 0
		sr, err = api.MessagesSearch(ctx, searchReq)
		if err != nil {
			return 0, err
		}
		cand = extractFirstMessage(sr)
		if cand == nil {
			return 0, nil
		}
	}
	parentTime := time.Unix(int64(parentDate), 0)
	candTime := time.Unix(int64(cand.Date), 0)
	if candTime.Sub(parentTime) > 20*time.Minute {
		return 0, nil
	}
	return cand.ID, nil
}

// extractFirstMessage извлекает первое сообщение из ответа поиска Telegram.
func extractFirstMessage(res tg.MessagesMessagesClass) *tg.Message {
	switch m := res.(type) {
	case *tg.MessagesMessages:
		if len(m.Messages) > 0 {
			if msg, ok := m.Messages[0].(*tg.Message); ok {
				return msg
			}
		}
	case *tg.MessagesMessagesSlice:
		if len(m.Messages) > 0 {
			if msg, ok := m.Messages[0].(*tg.Message); ok {
				return msg
			}
		}
	case *tg.MessagesChannelMessages:
		if len(m.Messages) > 0 {
			if msg, ok := m.Messages[0].(*tg.Message); ok {
				return msg
			}
		}
	}
	return nil
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

		// Определяем, нужно ли публиковать сообщение как ответ
		var replyTo *int
		if id, err := findReplyTarget(ctx, api, info.donor, info.target, msg); err != nil {
			log.Printf("[CHANNEL DUPLICATE] поиск сообщения для ответа: %v", err)
		} else if id != 0 {
			replyTo = &id
		}

		// Проверяем, есть ли в исходном сообщении предпросмотр ссылки
		_, hasPreview := msg.Media.(*tg.MessageMediaWebPage)

		// Исходный текст и сущности
		text := msg.Message
		entities := msg.Entities

		// Удаляем указанный фрагмент из текста
		if info.remove != nil && *info.remove != "" {
			if strings.Contains(text, *info.remove) {
				text = strings.ReplaceAll(text, *info.remove, "")
				entities = adjustEntitiesAfterRemoval(entities, msg.Message, *info.remove)
			}
		}

		baseText := text
		linkDetected := !hasPreview && hasURL(baseText)

		// Добавляем текст в конец поста, если это разрешено
		if info.add != nil && *info.add != "" && !hasPreview {
			addText := *info.add
			if ent, clean := parseTextURL(addText, utf16Len(baseText)); ent != nil {
				text = baseText + clean
				entities = append(entities, ent)
				linkDetected = true
			} else {
				text = baseText + addText
				if hasURL(addText) {
					linkDetected = true
				}
			}
		}

		disablePreview := linkDetected

		if err := copyMessage(ctx, api, info.donor, info.target, msg, text, entities, disablePreview, replyTo); err != nil {
			saveErr := db.SaveSos(fmt.Sprintf("не удалось скопировать пост %d с канала %d: %v", msg.ID, info.donor.ID, err))
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
