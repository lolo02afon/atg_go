package channel_duplicate

import (
	"regexp"
	"unicode/utf16"

	"github.com/gotd/td/tg"
)

// mdLinkPattern распознаёт Markdown-ссылку вида [текст](URL).
// Используем индексы для вычисления смещения и сохранения остальных частей строки.
var mdLinkPattern = regexp.MustCompile(`\[(.+?)\]\((https?://[^\s)]+)\)`)

// urlPattern ищет в тексте ссылку с протоколом http или https
var urlPattern = regexp.MustCompile(`https?://`)

// utf16Len возвращает длину строки в кодовых единицах UTF-16.
func utf16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}

// parseTextURL извлекает из текста Markdown-ссылку и формирует сущность Telegram.
// offset — смещение начала текста в UTF-16 относительно всей строки сообщения.
// Возвращается сущность ссылки и строка без Markdown-разметки.
func parseTextURL(text string, offset int) (*tg.MessageEntityTextURL, string) {
	idx := mdLinkPattern.FindStringSubmatchIndex(text)
	if len(idx) != 6 {
		// Возвращаем исходный текст, если ссылка не распознана
		return nil, text
	}
	// Извлекаем части строки до и после ссылки
	prefix := text[:idx[0]]
	label := text[idx[2]:idx[3]]
	url := text[idx[4]:idx[5]]
	suffix := text[idx[1]:]

	// Вычисляем смещение начала кликабельного текста
	ent := &tg.MessageEntityTextURL{
		Offset: offset + utf16Len(prefix),
		Length: utf16Len(label),
		URL:    url,
	}

	clean := prefix + label + suffix
	return ent, clean
}

// hasURL проверяет, содержит ли строка URL с протоколом http/https
func hasURL(text string) bool {
	return urlPattern.MatchString(text)
}
