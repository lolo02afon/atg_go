package channel_duplicate

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"atg_go/pkg/storage"
	base "atg_go/pkg/telegram/module"

	"github.com/gotd/td/tg"
	"github.com/lib/pq"
)

// channelInfo хранит данные для пересылки сообщений.
type channelInfo struct {
	id     int            // ID записи channel_duplicate
	donor  *tg.Channel    // Канал-источник
	target *tg.Channel    // Наш канал
	remove *string        // Текст для удаления из поста
	add    *string        // Текст для добавления
	skip   postSkip       // Условия пропуска постов
	lastID *int           // ID последнего обработанного поста
	times  pq.StringArray // Времена публикаций в формате HH:MM:SS
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

// processMessage подготавливает текст и публикует сообщение в целевом канале.
// Все проверки и модификации текста объединены здесь, чтобы избежать дублирования кода.
func processMessage(ctx context.Context, api *tg.Client, info channelInfo, msg *tg.Message) error {
	// Пропускаем рекламу
	if isAdvertisement(msg) {
		return nil
	}
	// Проверяем условия пропуска
	if shouldSkip(msg, info.skip) {
		return nil
	}

	// Подбираем сообщение для ответа
	var replyTo *int
	if id, err := findReplyTarget(ctx, api, info.donor, info.target, msg); err != nil {
		log.Printf("[CHANNEL DUPLICATE] поиск сообщения для ответа: %v", err)
	} else if id != 0 {
		replyTo = &id
	}

	// Проверяем наличие предпросмотра ссылки
	_, hasPreview := msg.Media.(*tg.MessageMediaWebPage)

	// Исходный текст и сущности
	text := msg.Message
	entities := msg.Entities

	// Удаляем указанный фрагмент
	if info.remove != nil && *info.remove != "" {
		if strings.Contains(text, *info.remove) {
			text = strings.ReplaceAll(text, *info.remove, "")
			entities = adjustEntitiesAfterRemoval(entities, msg.Message, *info.remove)
		}
	}

	baseText := text
	linkDetected := !hasPreview && hasURL(baseText)

	// Добавляем текст в конец поста
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

	return copyMessage(ctx, api, info.donor, info.target, msg, text, entities, disablePreview, replyTo)
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
		// Даже если пока нет записей, запускаем обработчик и слушатель
		log.Printf("[CHANNEL DUPLICATE] нет каналов для дублирования")
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

		if len(info.times) > 0 {
			// Для каналов с расписанием публикуем посты только по таймерам
			log.Printf("[CHANNEL DUPLICATE] пост %d из канала %d отложен согласно расписанию", msg.ID, peer.ChannelID)
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
		// Обновляем данные в карте
		info.remove = remove
		info.add = add
		info.lastID = &msg.ID
		chMap[peer.ChannelID] = info

		if err := processMessage(ctx, api, info, msg); err != nil {
			saveErr := db.SaveSos(fmt.Sprintf("не удалось скопировать пост %d с канала %d: %v", msg.ID, info.donor.ID, err))
			if saveErr != nil {
				log.Printf("[CHANNEL DUPLICATE] ошибка записи в Sos: %v", saveErr)
			}
		}
		return nil
	})

	for _, cd := range dups {
		registerDuplicate(ctx, api, db, accountID, cd, chMap)
	}

	// Запускаем слушатель событий БД для обновления списка каналов без перезапуска сервера
	go listenChannelDuplicate(ctx, api, db, accountID, chMap)
}

// registerDuplicate добавляет канал-донор в карту для отслеживания и подписывается на необходимые каналы.
func registerDuplicate(ctx context.Context, api *tg.Client, db *storage.DB, accountID int, cd storage.ChannelDuplicateOrder, chMap map[int64]channelInfo) {
	donorUser, err := base.Modf_ExtractUsername(cd.URLChannelDonor)
	if err != nil {
		log.Printf("[CHANNEL DUPLICATE] некорректная ссылка %s: %v", cd.URLChannelDonor, err)
		return
	}
	donorResolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: donorUser})
	if err != nil {
		log.Printf("[CHANNEL DUPLICATE] не удалось получить канал %s: %v", cd.URLChannelDonor, err)
		return
	}
	donorCh, err := base.Modf_FindChannel(donorResolved.GetChats())
	if err != nil {
		log.Printf("[CHANNEL DUPLICATE] канал %s не найден: %v", cd.URLChannelDonor, err)
		return
	}
	if err := base.Modf_JoinChannel(ctx, api, donorCh, db, accountID); err != nil && !strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
		log.Printf("[CHANNEL DUPLICATE] подписка на %s: %v", cd.URLChannelDonor, err)
		return
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
	donorIDStr := fmt.Sprintf("%d", donorCh.ID)
	if cd.ChannelDonorTGID == nil || *cd.ChannelDonorTGID != donorIDStr {
		_ = db.SetChannelDonorTGID(cd.ID, donorIDStr)
	}

	targetUser, err := base.Modf_ExtractUsername(cd.OrderURL)
	if err != nil {
		log.Printf("[CHANNEL DUPLICATE] некорректная ссылка %s: %v", cd.OrderURL, err)
		return
	}
	targetResolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: targetUser})
	if err != nil {
		log.Printf("[CHANNEL DUPLICATE] не удалось получить канал %s: %v", cd.OrderURL, err)
		return
	}
	targetCh, err := base.Modf_FindChannel(targetResolved.GetChats())
	if err != nil {
		log.Printf("[CHANNEL DUPLICATE] канал %s не найден: %v", cd.OrderURL, err)
		return
	}
	if err := base.Modf_JoinChannel(ctx, api, targetCh, db, accountID); err != nil && !strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
		log.Printf("[CHANNEL DUPLICATE] подписка на %s: %v", cd.OrderURL, err)
	}
	targetIDStr := fmt.Sprintf("%d", targetCh.ID)
	if cd.OrderChannelTGID == nil || *cd.OrderChannelTGID != targetIDStr {
		_ = db.SetOrderChannelTGID(cd.OrderID, targetIDStr)
	}

	// Удаляем старую запись, если ID канала изменился
	for key, info := range chMap {
		if info.id == cd.ID {
			delete(chMap, key)
			break
		}
	}
	chMap[donorCh.ID] = channelInfo{
		id:     cd.ID,
		donor:  donorCh,
		target: targetCh,
		remove: cd.PostTextRemove,
		add:    cd.PostTextAdd,
		skip:   parsePostSkip(cd.PostSkip),
		lastID: cd.LastPostID,
		times:  cd.PostCountDay,
	}

	if cd.LastPostID == nil {
		// В БД всегда ожидается last_post_id; при его отсутствии выходим
		log.Printf("[CHANNEL DUPLICATE] для канала %d отсутствует last_post_id", donorCh.ID)
		return
	}

	if len(cd.PostCountDay) > 0 {
		// При наличии расписания запускаем публикацию по временным меткам
		log.Printf("[CHANNEL DUPLICATE] обнаружены last_post_id %d и расписание %v для канала %d", *cd.LastPostID, cd.PostCountDay, donorCh.ID)
		go postFromHistory(ctx, api, db, donorCh.ID, chMap)
	} else {
		// Без расписания сразу публикуем все пропущенные посты
		log.Printf("[CHANNEL DUPLICATE] обнаружен last_post_id %d без расписания для канала %d", *cd.LastPostID, donorCh.ID)
		go postFromHistoryImmediate(ctx, api, db, donorCh.ID, chMap)
	}
}

// postFromHistory планирует публикации постов из истории донорского канала по указанным времени.
// Для каждого времени создаётся отдельная горутина, которая раз в сутки публикует следующий пост.
func postFromHistory(ctx context.Context, api *tg.Client, db *storage.DB, donorID int64, chMap map[int64]channelInfo) {
	info, ok := chMap[donorID]
	if !ok {
		log.Printf("[CHANNEL DUPLICATE] канал %d не найден для запуска публикации из истории", donorID)
		return
	}
	if info.lastID == nil {
		log.Printf("[CHANNEL DUPLICATE] для канала %d отсутствует last_post_id", donorID)
		return
	}
	if len(info.times) == 0 {
		log.Printf("[CHANNEL DUPLICATE] для канала %d не заданы времена публикации", donorID)
		return
	}

	log.Printf("[CHANNEL DUPLICATE] запуск таймеров для канала %d: %v", donorID, info.times)

	for _, tStr := range info.times {
		tStr = strings.TrimSpace(tStr)
		parsed, err := time.Parse("15:04:05", tStr)
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] неверный формат времени %q для канала %d: %v", tStr, donorID, err)
			continue
		}
		hour, minute, second := parsed.Hour(), parsed.Minute(), parsed.Second()
		log.Printf("[CHANNEL DUPLICATE] планируем публикацию в %02d:%02d:%02d для канала %d", hour, minute, second, donorID)

		// Запускаем отдельную горутину для каждого времени публикации
		go func(h, m, s int, original string) {
			for {
				now := time.Now()
				next := time.Date(now.Year(), now.Month(), now.Day(), h, m, s, 0, now.Location())
				if !next.After(now) {
					next = next.Add(24 * time.Hour)
				}

				log.Printf("[CHANNEL DUPLICATE] ожидание до %s для канала %d (%s)", next.Format(time.RFC3339), donorID, original)
				timer := time.NewTimer(next.Sub(now))
				select {
				case <-ctx.Done():
					log.Printf("[CHANNEL DUPLICATE] остановка таймера для канала %d (%s)", donorID, original)
					timer.Stop()
					return
				case <-timer.C:
					log.Printf("[CHANNEL DUPLICATE] наступило время %s для канала %d", next.Format("15:04:05"), donorID)
					publishNextFromHistory(ctx, api, db, donorID, chMap)
				}
			}
		}(hour, minute, second, tStr)
	}
}

// postFromHistoryImmediate публикует все пропущенные посты сразу.
// Используется, когда post_count_day равен NULL.
func postFromHistoryImmediate(ctx context.Context, api *tg.Client, db *storage.DB, donorID int64, chMap map[int64]channelInfo) {
	info, ok := chMap[donorID]
	if !ok || info.lastID == nil {
		return
	}

	// Получаем ID последнего поста на донорском канале
	posts, err := base.GetChannelPosts(ctx, api, info.donor, 1)
	if err != nil {
		log.Printf("[CHANNEL DUPLICATE] получение последнего поста канала %d: %v", donorID, err)
		return
	}
	lastDonorID := posts[0].ID

	// Если сохранённый ID уже соответствует последнему посту, публикация не требуется
	if *info.lastID >= lastDonorID {
		return
	}

	// Перебираем все ID от сохранённого до последнего поста включительно
	for id := *info.lastID + 1; id <= lastDonorID; id++ {
		publishPostByID(ctx, api, db, donorID, chMap, id)
	}
}

// publishPostByID публикует указанный пост из истории, если он существует.
// Используется только при отсутствии расписания публикаций.
func publishPostByID(ctx context.Context, api *tg.Client, db *storage.DB, donorID int64, chMap map[int64]channelInfo, postID int) {
	info, ok := chMap[donorID]
	if !ok {
		log.Printf("[CHANNEL DUPLICATE] нет информации о канале %d при публикации поста %d", donorID, postID)
		return
	}

	res, err := api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
		Channel: &tg.InputChannel{ChannelID: info.donor.ID, AccessHash: info.donor.AccessHash},
		ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: postID}},
	})
	if err != nil {
		log.Printf("[CHANNEL DUPLICATE] получение поста %d: %v", postID, err)
		return
	}

	var msg *tg.Message
	switch m := res.(type) {
	case *tg.MessagesChannelMessages:
		if len(m.Messages) > 0 {
			if mm, ok := m.Messages[0].(*tg.Message); ok {
				msg = mm
			}
		}
	case *tg.MessagesMessages:
		if len(m.Messages) > 0 {
			if mm, ok := m.Messages[0].(*tg.Message); ok {
				msg = mm
			}
		}
	}
	if msg == nil {
		log.Printf("[CHANNEL DUPLICATE] пост %d не найден в канале %d", postID, donorID)
		return
	}

	updated, remove, add, err := db.TrySetLastPostID(info.id, msg.ID)
	if err != nil {
		log.Printf("[CHANNEL DUPLICATE] обновление last_post_id: %v", err)
		return
	}

	info.lastID = &msg.ID
	chMap[donorID] = info
	if !updated {
		log.Printf("[CHANNEL DUPLICATE] пост %d уже был опубликован для канала %d", msg.ID, donorID)
		return
	}
	info.remove = remove
	info.add = add
	chMap[donorID] = info

	if err := processMessage(ctx, api, info, msg); err != nil {
		saveErr := db.SaveSos(fmt.Sprintf("не удалось скопировать пост %d с канала %d: %v", msg.ID, info.donor.ID, err))
		if saveErr != nil {
			log.Printf("[CHANNEL DUPLICATE] ошибка записи в Sos: %v", saveErr)
		}
	} else {
		log.Printf("[CHANNEL DUPLICATE] опубликован пост %d для канала %d", msg.ID, donorID)
	}
}

// publishNextFromHistory берёт следующий пост после last_post_id и публикует его в целевой канал.
func publishNextFromHistory(ctx context.Context, api *tg.Client, db *storage.DB, donorID int64, chMap map[int64]channelInfo) {
	info, ok := chMap[donorID]
	if !ok {
		log.Printf("[CHANNEL DUPLICATE] нет информации о канале %d при публикации из истории", donorID)
		return
	}
	if info.lastID == nil {
		log.Printf("[CHANNEL DUPLICATE] отсутствует last_post_id для канала %d", donorID)
		return
	}

	currentID := *info.lastID
	log.Printf("[CHANNEL DUPLICATE] публикация следующего поста после %d для канала %d", currentID, donorID)
	// Ограничиваем число попыток до 50, чтобы не попасть в бесконечный цикл при пропущенных ID
	for attempts := 0; attempts < 50; attempts++ {
		currentID++
		log.Printf("[CHANNEL DUPLICATE] попытка %d: ищем пост %d для канала %d", attempts+1, currentID, donorID)
		res, err := api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
			Channel: &tg.InputChannel{ChannelID: info.donor.ID, AccessHash: info.donor.AccessHash},
			ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: currentID}},
		})
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] получение поста %d: %v", currentID, err)
			return
		}

		var msg *tg.Message
		switch m := res.(type) {
		case *tg.MessagesChannelMessages:
			if len(m.Messages) > 0 {
				if mm, ok := m.Messages[0].(*tg.Message); ok {
					msg = mm
				}
			}
		case *tg.MessagesMessages:
			if len(m.Messages) > 0 {
				if mm, ok := m.Messages[0].(*tg.Message); ok {
					msg = mm
				}
			}
		}
		if msg == nil {
			log.Printf("[CHANNEL DUPLICATE] пост %d не найден в канале %d", currentID, donorID)
			continue
		}

		updated, remove, add, err := db.TrySetLastPostID(info.id, msg.ID)
		if err != nil {
			log.Printf("[CHANNEL DUPLICATE] обновление last_post_id: %v", err)
			return
		}

		info.lastID = &msg.ID
		chMap[donorID] = info
		if !updated {
			log.Printf("[CHANNEL DUPLICATE] пост %d уже был опубликован для канала %d", msg.ID, donorID)
			continue
		}
		info.remove = remove
		info.add = add
		chMap[donorID] = info

		if err := processMessage(ctx, api, info, msg); err != nil {
			saveErr := db.SaveSos(fmt.Sprintf("не удалось скопировать пост %d с канала %d: %v", msg.ID, info.donor.ID, err))
			if saveErr != nil {
				log.Printf("[CHANNEL DUPLICATE] ошибка записи в Sos: %v", saveErr)
			}
		} else {
			log.Printf("[CHANNEL DUPLICATE] опубликован пост %d для канала %d", msg.ID, donorID)
		}
		break
	}
}

// listenChannelDuplicate ожидает уведомления от PostgreSQL об изменениях в таблице channel_duplicate.
func listenChannelDuplicate(ctx context.Context, api *tg.Client, db *storage.DB, accountID int, chMap map[int64]channelInfo) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/atg_db?sslmode=disable&application_name=atg_app"
	}
	listener := pq.NewListener(dsn, 10*time.Second, time.Minute, nil)
	if err := listener.Listen("channel_duplicate_changed"); err != nil {
		log.Printf("[CHANNEL DUPLICATE] подписка на уведомления: %v", err)
		return
	}
	defer listener.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case n := <-listener.Notify:
			if n == nil {
				continue
			}
			id, err := strconv.Atoi(n.Extra)
			if err != nil {
				log.Printf("[CHANNEL DUPLICATE] некорректный payload %q: %v", n.Extra, err)
				continue
			}
			cd, err := db.GetChannelDuplicateOrderByID(id)
			if err != nil {
				if err == sql.ErrNoRows {
					// Запись удалена — убираем её из карты
					for key, info := range chMap {
						if info.id == id {
							delete(chMap, key)
							break
						}
					}
					continue
				}
				log.Printf("[CHANNEL DUPLICATE] получение записи %d: %v", id, err)
				continue
			}
			registerDuplicate(ctx, api, db, accountID, *cd, chMap)
		}
	}
}
