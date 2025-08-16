package module

import (
	"context"
	"log"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"

	"github.com/gotd/td/tg"
)

// ModF_UnsubscribeAll отключает указанное количество каналов и групп у всех аккаунтов.
func ModF_UnsubscribeAll(db *storage.DB, delay [2]int, limit int) error {
	rand.Seed(time.Now().UnixNano())

	// Получаем ссылки из заказов один раз, чтобы не обращаться к БД при каждой отписке
	orderLinks, err := db.GetOrdersDefaultURLs()
	if err != nil {
		return err
	}

	// Преобразуем ссылки в множество имён каналов
	skipChannels := make(map[string]struct{})
	for _, l := range orderLinks {
		if name, ok := channelUsernameFromLink(l); ok {
			skipChannels[strings.ToLower(name)] = struct{}{}
		}
	}

	accounts, err := db.GetAuthorizedAccounts()
	if err != nil {
		return err
	}
	for _, acc := range accounts {
		log.Printf("[UNSUBSCRIBE] аккаунт %d: начало обработки", acc.ID)
		if err := unsubscribeAccount(db, &acc, delay, limit, skipChannels); err != nil {
			log.Printf("[UNSUBSCRIBE] аккаунт %d: %v", acc.ID, err)
		} else {
			log.Printf("[UNSUBSCRIBE] аккаунт %d: завершено", acc.ID)
		}
	}
	return nil
}

// unsubscribeAccount выходит из указанного количества каналов и групп для одного аккаунта.
func unsubscribeAccount(db *storage.DB, acc *models.Account, delay [2]int, limit int, skip map[string]struct{}) error {
	client, err := Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)
		res, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			Limit:      100,
			OffsetPeer: &tg.InputPeerEmpty{},
		})
		if err != nil {
			return err
		}
		dialogs, ok := res.AsModified()
		if !ok {
			return nil
		}

		// Собираем ID обсуждений для каналов, отписка от которых запрещена
		bannedChats := make(map[int64]struct{})
		for _, ch := range dialogs.GetChats() {
			if c, ok := ch.(*tg.Channel); ok {
				if _, banned := skip[strings.ToLower(c.Username)]; banned {
					if c.LinkedChatID != 0 {
						bannedChats[c.LinkedChatID] = struct{}{}
					}
				}
			}
		}

		count := 0
		for _, raw := range dialogs.GetDialogs() {
			if count >= limit {
				break
			}
			d, ok := raw.(*tg.Dialog)
			if !ok {
				continue
			}
			switch peer := d.Peer.(type) {
			case *tg.PeerChannel:
				if ch := findChannel(dialogs.GetChats(), peer.ChannelID); ch != nil {
					if _, banned := skip[strings.ToLower(ch.Username)]; banned {
						continue
					}
					time.Sleep(randomDelay(delay))
					if _, err := api.ChannelsLeaveChannel(ctx, &tg.InputChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}); err != nil {
						log.Printf("[UNSUBSCRIBE] аккаунт %d не покинул канал %d: %v", acc.ID, ch.ID, err)
					} else {
						log.Printf("[UNSUBSCRIBE] аккаунт %d покинул канал %d (%s)", acc.ID, ch.ID, channelAddress(ch))
						count++
					}
				}
			case *tg.PeerChat:
				if _, banned := bannedChats[peer.ChatID]; banned {
					continue
				}
				time.Sleep(randomDelay(delay))
				if _, err := api.MessagesDeleteChatUser(ctx, &tg.MessagesDeleteChatUserRequest{ChatID: peer.ChatID, UserID: &tg.InputUserSelf{}}); err != nil {
					log.Printf("[UNSUBSCRIBE] аккаунт %d не покинул группу %d: %v", acc.ID, peer.ChatID, err)
				} else {
					log.Printf("[UNSUBSCRIBE] аккаунт %d покинул группу %d", acc.ID, peer.ChatID)
					count++
				}
			}
		}
		return nil
	})
}

// findChannel ищет канал по идентификатору среди чатов.
func findChannel(chats []tg.ChatClass, id int64) *tg.Channel {
	for _, ch := range chats {
		if c, ok := ch.(*tg.Channel); ok && c.ID == id {
			return c
		}
	}
	return nil
}

// channelAddress формирует ссылку на канал или возвращает заглушку, если адреса нет.
func channelAddress(ch *tg.Channel) string {
	if ch.Username != "" {
		return "https://t.me/" + ch.Username
	}
	return "адрес недоступен"
}

// channelUsernameFromLink извлекает имя канала из URL, если это корректная ссылка на Telegram-канал
func channelUsernameFromLink(link string) (string, bool) {
	if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
		link = "https://" + link
	}
	u, err := url.Parse(link)
	if err != nil {
		return "", false
	}
	host := strings.ToLower(u.Host)
	if host != "t.me" && host != "telegram.me" {
		return "", false
	}
	name := strings.Trim(u.Path, "/")
	if name == "" || strings.Contains(name, "/") || strings.HasPrefix(name, "+") || strings.HasPrefix(name, "joinchat") || strings.HasPrefix(name, "c/") {
		return "", false
	}
	return name, true
}

// randomDelay возвращает случайную задержку в заданном диапазоне.
func randomDelay(r [2]int) time.Duration {
	if r[1] <= r[0] {
		return time.Duration(r[0]) * time.Second
	}
	d := rand.Intn(r[1]-r[0]+1) + r[0]
	return time.Duration(d) * time.Second
}
