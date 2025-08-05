package module

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

// GetDiscussionReplies возвращает последние сообщения в обсуждении поста.
func GetDiscussionReplies(ctx context.Context, api *tg.Client, chat *tg.Channel, msgID, limit int) ([]*tg.Message, error) {
	res, err := api.MessagesGetReplies(ctx, &tg.MessagesGetRepliesRequest{
		Peer:  &tg.InputPeerChannel{ChannelID: chat.ID, AccessHash: chat.AccessHash},
		MsgID: msgID,
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	msgs, ok := res.(*tg.MessagesChannelMessages)
	if !ok {
		return nil, fmt.Errorf("unexpected messages type")
	}

	var valid []*tg.Message
	for _, m := range msgs.Messages {
		if msg, ok := m.(*tg.Message); ok {
			valid = append(valid, msg)
		}
	}

	if len(valid) == 0 {
		return nil, fmt.Errorf("no valid messages")
	}

	return valid, nil
}
