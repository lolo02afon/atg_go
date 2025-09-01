package channel_duplicate

import (
	"encoding/json"
	"strings"

	"github.com/gotd/td/tg"
)

// postSkip задаёт условия, при которых сообщение не публикуется.
// text - строки, которые запрещены в тексте поста;
// url  - строки, которые запрещены в адресах ссылок.
type postSkip struct {
	Text []string `json:"text"`
	URL  []string `json:"url"`
}

// parsePostSkip разбирает JSON с условиями пропуска.
// При ошибке возвращает пустую структуру, чтобы не блокировать дублирование.
func parsePostSkip(data []byte) postSkip {
	if len(data) == 0 {
		return postSkip{}
	}
	var ps postSkip
	if err := json.Unmarshal(data, &ps); err != nil {
		return postSkip{}
	}
	return ps
}

// shouldSkip проверяет, удовлетворяет ли сообщение условиям пропуска.
// Возвращает true, если сообщение нужно пропустить.
func shouldSkip(msg *tg.Message, ps postSkip) bool {
	lowerText := strings.ToLower(msg.Message)
	for _, t := range ps.Text {
		if t == "" {
			continue
		}
		if strings.Contains(lowerText, strings.ToLower(t)) {
			return true
		}
	}
	if len(ps.URL) == 0 {
		return false
	}
	links := extractLinks(msg)
	for _, l := range links {
		l = strings.ToLower(l)
		for _, u := range ps.URL {
			if u == "" {
				continue
			}
			if strings.Contains(l, strings.ToLower(u)) {
				return true
			}
		}
	}
	return false
}
