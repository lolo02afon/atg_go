package channel_duplicate

import (
	"regexp"
	"unicode/utf16"

	"github.com/gotd/td/tg"
)

// mdLinkPattern распознаёт Markdown-ссылку вида [текст](URL).
var mdLinkPattern = regexp.MustCompile(`\[(.+?)\]\((https?://[^\s)]+)\)`)

// utf16Len возвращает длину строки в кодовых единицах UTF-16.
func utf16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}

// parseTextURL извлекает из текста Markdown-ссылку и формирует сущность Telegram.
// offset — смещение начала текста в UTF-16 относительно всей строки сообщения.
func parseTextURL(text string, offset int) (*tg.MessageEntityTextURL, string) {
	match := mdLinkPattern.FindStringSubmatch(text)
	if len(match) != 3 {
		// Возвращаем исходный текст, если ссылка не распознана
		return nil, text
	}
	label := match[1]
	url := match[2]
	ent := &tg.MessageEntityTextURL{
		Offset: offset,
		Length: utf16Len(label),
		URL:    url,
	}
	return ent, label
}
