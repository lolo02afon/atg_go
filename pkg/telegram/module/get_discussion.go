package module

import (
	"context"
	"fmt"

	"atg_go/pkg/storage"

	"github.com/gotd/td/tg"
)

// Discussion содержит канал обсуждения и все ответы под конкретным постом
type Discussion struct {
	Chat        *tg.Channel
	PostMessage *tg.Message // сообщение, соответствующее посту в канале
	Replies     []*tg.Message
}

// Modf_getPostDiscussion получает обсуждение для указанного сообщения (поста) в канале.
func Modf_getPostDiscussion(
	ctx context.Context,
	api *tg.Client,
	channel *tg.Channel,
	msgID int,
) (*Discussion, error) {

	// Запрашиваем информацию об обсуждении
	discussMsg, err := api.MessagesGetDiscussionMessage(ctx, &tg.MessagesGetDiscussionMessageRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		},
		MsgID: msgID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get discussion: %w", err)
	}

	// 1) выбираем linkedChat — чат обсуждения, отличный от основного канала
	var linkedChat *tg.Channel
	for _, raw := range discussMsg.GetChats() {
		if ch, ok := raw.(*tg.Channel); ok && ch.ID != channel.ID {
			linkedChat = ch
			break
		}
	}
	if linkedChat == nil {
		return nil, fmt.Errorf("discussion chat not found")
	}

	// 2) разбираем сообщения из ответа, выбираем корневое сообщение обсуждения (которое находится в связанном чате) и остальные сообщения считаем ответами
	var (
		postMsg *tg.Message
		replies []*tg.Message
	)
	for _, raw := range discussMsg.GetMessages() {
		m, ok := raw.(*tg.Message)
		if !ok {
			continue
		}
		peer, ok := m.PeerID.(*tg.PeerChannel)
		if !ok || peer.ChannelID != linkedChat.ID {
			// Нас интересуют только сообщения из чата обсуждения
			continue
		}

		if m.ReplyTo == nil {
			// сообщение без ReplyTo — корневой пост обсуждения
			postMsg = m
			continue
		}

		replies = append(replies, m)
	}
	if postMsg == nil {
		return nil, fmt.Errorf("discussion post message not found")
	}

	return &Discussion{
		Chat:        linkedChat,
		PostMessage: postMsg,
		Replies:     replies,
	}, nil
}

// Modf_getDiscussionChat возвращает чат обсуждения, связанный с каналом.
// Функция не привязана к конкретному посту и просто ищет связанный чат.
func Modf_getDiscussionChat(ctx context.Context, api *tg.Client, channel *tg.Channel, db *storage.DB, accountID int) (*tg.Channel, error) {
	// Запрашиваем полную информацию о канале, чтобы узнать ID связанного чата
	full, err := api.ChannelsGetFullChannel(ctx, &tg.InputChannel{
		ChannelID:  channel.ID,
		AccessHash: channel.AccessHash,
	})
	if err != nil {
		return nil, fmt.Errorf("не удалось получить данные канала: %w", err)
	}

	fullChat, ok := full.GetFullChat().(*tg.ChannelFull)
	if !ok {
		return nil, fmt.Errorf("неожиданный тип полного канала")
	}
	if fullChat.LinkedChatID == 0 {
		return nil, fmt.Errorf("у канала нет чата обсуждения")
	}

	// Извлекаем объект обсуждения из full.GetChats, чтобы получить access hash
	var discussion *tg.Channel
	for _, raw := range full.GetChats() {
		if ch, ok := raw.(*tg.Channel); ok && ch.ID == fullChat.LinkedChatID {
			discussion = ch
			break
		}
	}
	if discussion == nil {
		return nil, fmt.Errorf("не удалось найти чат обсуждения")
	}

	// Получаем подробную информацию о чате обсуждения
	chats, err := api.ChannelsGetChannels(ctx, []tg.InputChannelClass{&tg.InputChannel{ChannelID: discussion.ID, AccessHash: discussion.AccessHash}})
	if err != nil {
		if tg.IsChannelPrivate(err) || tg.IsChannelParicipantMissing(err) {
			// Присоединяемся к чату и пробуем ещё раз
			if errJoin := Modf_JoinChannel(ctx, api, discussion, db, accountID); errJoin != nil {
				return nil, fmt.Errorf("не удалось присоединиться к чату обсуждения: %w", errJoin)
			}

			chats, err = api.ChannelsGetChannels(ctx, []tg.InputChannelClass{&tg.InputChannel{ChannelID: discussion.ID, AccessHash: discussion.AccessHash}})
			if err != nil {
				return nil, fmt.Errorf("не удалось получить чат обсуждения после присоединения: %w", err)
			}
		} else {
			return nil, fmt.Errorf("не удалось получить чат обсуждения: %w", err)
		}
	}

	for _, raw := range chats.GetChats() {
		if ch, ok := raw.(*tg.Channel); ok && ch.ID == discussion.ID {
			return ch, nil
		}
	}

	return nil, fmt.Errorf("чат обсуждения не найден")
}
