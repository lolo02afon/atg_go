package channel_duplicate

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/gotd/td/tg"
)

// linkPattern ищет ссылки в тексте поста.
var linkPattern = regexp.MustCompile(`https?://\S+`)

// isAdvertisement определяет, содержит ли сообщение признаки рекламы.
func isAdvertisement(msg *tg.Message) bool {
	text := strings.ToLower(msg.Message)

	// Проверяем, что сообщение не переслано из другого канала
	if fwd, ok := msg.GetFwdFrom(); ok {
		// Forwarded из канала определяем по наличию ChannelPost
		if _, ok := fwd.GetChannelPost(); ok {
			return true
		}
		// или если источник указан как PeerChannel
		if from, ok := fwd.GetFromID(); ok {
			if _, ok := from.(*tg.PeerChannel); ok {
				return true
			}
		}
	}

	// Проверяем ключевые слова в тексте
	if strings.Contains(text, "erid") || strings.Contains(text, "ерид") {
		return true
	}
	if strings.Contains(text, "#реклама") || strings.Contains(text, "#sponsored") || strings.Contains(text, "#ad") {
		return true
	}

	// Проверяем все ссылки в сообщении
	links := extractLinks(msg)
	for _, l := range links {
		lower := strings.ToLower(l)
		if strings.Contains(lower, "erid") || strings.Contains(lower, "ерид") {
			return true
		}
		u, err := url.Parse(l)
		if err != nil {
			continue
		}
		switch strings.ToLower(u.Query().Get("utm_medium")) {
		case "influencer", "sponsor", "ed", "ads", "promo", "pr", "native":
			return true
		}
	}

	return false
}

// extractLinks возвращает все ссылки из сообщения.
func extractLinks(msg *tg.Message) []string {
	var links []string
	links = append(links, linkPattern.FindAllString(msg.Message, -1)...)
	for _, e := range msg.Entities {
		if t, ok := e.(*tg.MessageEntityTextURL); ok {
			links = append(links, t.URL)
		}
	}
	return links
}
