package module

import (
	"context"
	"fmt"

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
