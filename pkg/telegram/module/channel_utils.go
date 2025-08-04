package module

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

// подписывает аккаунт на указанный канал (или группу обсуждения).
func Modf_JoinChannel(ctx context.Context, api *tg.Client, channel *tg.Channel) error {
	// Передаём напрямую InputChannel — так реализован метод ChannelsJoinChannel в gotd/td
	_, err := api.ChannelsJoinChannel(ctx, &tg.InputChannel{
		ChannelID:  channel.ID,
		AccessHash: channel.AccessHash,
	})
	return err
}

// возвращает список сообщений из канала
func GetChannelPosts(ctx context.Context, api *tg.Client, channel *tg.Channel, limit int) ([]*tg.Message, error) {
	history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		},
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	messages, ok := history.(*tg.MessagesChannelMessages)
	if !ok {
		return nil, fmt.Errorf("unexpected messages type")
	}

	var validMessages []*tg.Message
	for _, msg := range messages.Messages {
		if m, ok := msg.(*tg.Message); ok {
			validMessages = append(validMessages, m)
		}
	}

	if len(validMessages) == 0 {
		return nil, fmt.Errorf("no valid messages")
	}

	return validMessages, nil
}

// извлекает username из URL канала
func Modf_ExtractUsername(url string) (string, error) {
	if !strings.HasPrefix(url, "https://t.me/") {
		return "", fmt.Errorf("invalid URL format")
	}
	return strings.TrimPrefix(url, "https://t.me/"), nil
}

// находит канал в списке чатов
func Modf_FindChannel(chats []tg.ChatClass) (*tg.Channel, error) {
	for _, peer := range chats {
		if ch, ok := peer.(*tg.Channel); ok {
			// Если канал является мегагруппой (обсуждением), пропускаем его
			if ch.Megagroup {
				continue
			}
			// Возвращаем первый найденный вещательный канал
			if ch.Broadcast {
				return ch, nil
			}
		}
	}
	return nil, fmt.Errorf("broadcast channel not found")
}

// выбирает случайный пост из канала
func Modf_GetRandomChannelPost(ctx context.Context, api *tg.Client, channel *tg.Channel, limit int) (*tg.Message, error) {
	history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		},
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	messages, ok := history.(*tg.MessagesChannelMessages)
	if !ok {
		return nil, fmt.Errorf("unexpected messages type")
	}

	var validMessages []*tg.Message
	for _, msg := range messages.Messages {
		if m, ok := msg.(*tg.Message); ok {
			validMessages = append(validMessages, m)
		}
	}

	if len(validMessages) == 0 {
		return nil, fmt.Errorf("no valid messages")
	}

	return validMessages[rand.Intn(len(validMessages))], nil
}

// Создаем клиент Telegram с указанными параметрами
func Modf_AccountInitialization(apiID int, apiHash, phone string) (*telegram.Client, error) {

	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{
			Path: "sessions/" + phone + ".session.json",
		},
	})

	return client, nil
}
