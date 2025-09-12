package module

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

// GetAllowedReactions возвращает список эмодзи, которые разрешены
// для использования в указанном канале или обсуждении.
// Если канал позволяет любые стандартные реакции, возвращается исходный
// список base. Если разрешённых реакций нет, возвращается пустой список.
func GetAllowedReactions(
	ctx context.Context,
	api *tg.Client,
	channel *tg.Channel,
	base []string,
) ([]string, error) {
	full, err := api.ChannelsGetFullChannel(ctx, &tg.InputChannel{
		ChannelID:  channel.ID,
		AccessHash: channel.AccessHash,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get full channel: %w", err)
	}

	fullChat, ok := full.GetFullChat().(*tg.ChannelFull)
	if !ok {
		// Не удалось получить полную информацию о канале
		return nil, fmt.Errorf("unexpected chat full type")
	}

	reactions, ok := fullChat.GetAvailableReactions()
	if !ok || reactions == nil {
		// Если нет информации — считаем, что разрешены все реакции из base
		return base, nil
	}

	switch r := reactions.(type) {
	case *tg.ChatReactionsAll:
		// Разрешены все стандартные реакции — используем наш базовый список
		return base, nil
	case *tg.ChatReactionsNone:
		// В канале отключены реакции
		return []string{}, nil
	case *tg.ChatReactionsSome:
		// Составляем пересечение разрешённых реакций и нашего базового списка
		allowed := make(map[string]struct{})
		for _, rc := range r.Reactions {
			if emoji, ok := rc.(*tg.ReactionEmoji); ok {
				allowed[emoji.Emoticon] = struct{}{}
			}
		}
		var result []string
		for _, e := range base {
			if _, ok := allowed[e]; ok {
				result = append(result, e)
			}
		}
		return result, nil
	default:
		// Неизвестный тип — возвращаем базовый список
		return base, nil
	}
}
