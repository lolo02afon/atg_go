package module

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strings"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	statistics "atg_go/pkg/telegram/invite_activities_statistics"

	"golang.org/x/net/proxy"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/tg"
)

// Modf_JoinChannel подписывает аккаунт на канал с учётом лимита подписок.
// Если ранее был достигнут предел в 500 каналов, новая попытка не выполняется до конца суток.
func Modf_JoinChannel(ctx context.Context, api *tg.Client, channel *tg.Channel, db *storage.DB, accountID int) error {
	// Проверяем, не активен ли блок на новые подписки для данного аккаунта.
	blocked, err := statistics.IsChannelsLimitActive(db, accountID)
	if err != nil {
		return fmt.Errorf("не удалось проверить лимит подписок: %w", err)
	}
	if blocked {
		// Если блок активен, не делаем запрос и сообщаем об этом через ошибку.
		return fmt.Errorf("аккаунт %d временно не может подписываться на новые каналы", accountID)
	}

	// Пытаемся подписаться на канал.
	_, err = api.ChannelsJoinChannel(ctx, &tg.InputChannel{
		ChannelID:  channel.ID,
		AccessHash: channel.AccessHash,
	})
	if err != nil {
		// Если ошибка связана с превышением лимита 500 каналов, фиксируем её и ставим блок до конца суток.
		if strings.Contains(err.Error(), "USER_CHANNELS_TOO_MUCH") {
			// Вычисляем момент времени 23:59 текущих суток.
			now := time.Now()
			until := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
			_ = statistics.MarkChannelsLimit(db, accountID, until) // Обновляем время блокировки в БД
			log.Printf("[WARN] Аккаунт %d достиг лимита подписок: %v", accountID, err)
		}
		return err
	}
	return nil
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

	// Сортируем сообщения по убыванию ID, чтобы новые были первыми
	sort.Slice(validMessages, func(i, j int) bool {
		return validMessages[i].ID > validMessages[j].ID
	})

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

// Создаем клиент Telegram с указанными параметрами и хранилищем сессии в БД.
func Modf_AccountInitialization(apiID int, apiHash, phone string, p *models.Proxy, r *rand.Rand, db *sql.DB, accountID int, h telegram.UpdateHandler) (*telegram.Client, error) {
	var storage session.Storage = &session.StorageMemory{}
	if db != nil && accountID > 0 {
		storage = &DBSessionStorage{DB: db, AccountID: accountID}
	}

	opts := telegram.Options{SessionStorage: storage}
	if h != nil {
		opts.UpdateHandler = h
	}
	if r != nil {
		opts.Random = r
	}
	if p != nil {
		addr := fmt.Sprintf("%s:%d", p.IP, p.Port)
		var auth *proxy.Auth
		if p.Login != "" || p.Password != "" {
			auth = &proxy.Auth{User: p.Login, Password: p.Password}
		}
		d, err := proxy.SOCKS5("tcp", addr, auth, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("proxy dialer: %w", err)
		}
		dc, ok := d.(proxy.ContextDialer)
		if !ok {
			return nil, fmt.Errorf("proxy dialer missing context")
		}
		opts.Resolver = dcs.Plain(dcs.PlainOptions{Dial: dc.DialContext})
		log.Printf("[PROXY] %s via %s", phone, addr)
	}
	client := telegram.NewClient(apiID, apiHash, opts)
	return client, nil
}
