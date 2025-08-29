package module

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

// defaultReactions содержит список реакций, используемых, если канал разрешает все стандартные
// реакции и не предоставил свой собственный список.
var defaultReactions = []string{"👍", "❤️", "🔥", "😁"}

// GetChannelReactions возвращает список реакций, разрешённых каналом.
// Возвращаются не более limit реакций в том порядке, в котором их предоставляет канал.
// Если у канала отключены реакции, возвращается ошибка.
func GetChannelReactions(ctx context.Context, api *tg.Client, channel *tg.Channel, limit int) ([]string, error) {
	full, err := api.ChannelsGetFullChannel(ctx, &tg.InputChannel{ChannelID: channel.ID, AccessHash: channel.AccessHash})
	if err != nil {
		return nil, fmt.Errorf("не удалось получить данные канала: %w", err)
	}

	fullChat, ok := full.GetFullChat().(*tg.ChannelFull)
	if !ok {
		return nil, fmt.Errorf("неизвестный тип полной информации о канале")
	}

	reactions, ok := fullChat.GetAvailableReactions()
	if !ok || reactions == nil {
		// Если канал не предоставляет список реакций, используем набор по умолчанию
		if len(defaultReactions) < limit {
			return defaultReactions, nil
		}
		return defaultReactions[:limit], nil
	}

	switch r := reactions.(type) {
	case *tg.ChatReactionsAll:
		// Канал разрешает все реакции, используем набор по умолчанию
		if len(defaultReactions) < limit {
			return defaultReactions, nil
		}
		return defaultReactions[:limit], nil
	case *tg.ChatReactionsNone:
		return nil, fmt.Errorf("в канале отключены реакции")
	case *tg.ChatReactionsSome:
		var result []string
		for _, rc := range r.Reactions {
			if len(result) >= limit {
				break
			}
			if emoji, ok := rc.(*tg.ReactionEmoji); ok {
				result = append(result, emoji.Emoticon)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("неизвестный тип списка реакций")
	}
}
