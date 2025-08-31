package channel_duplicate

import "testing"

// TestParseTextURLSimple проверяет извлечение ссылки без окружения.
func TestParseTextURLSimple(t *testing.T) {
	base := "😀Привет" // emoji занимает две позиции UTF-16
	add := "[Мир](https://example.com)"
	ent, clean := parseTextURL(add, utf16Len(base))
	if ent == nil {
		t.Fatalf("сущность не найдена")
	}
	if ent.Offset != utf16Len(base) {
		t.Errorf("ожидался offset %d, получен %d", utf16Len(base), ent.Offset)
	}
	if ent.Length != utf16Len("Мир") {
		t.Errorf("ожидалась длина %d, получена %d", utf16Len("Мир"), ent.Length)
	}
	if clean != "Мир" {
		t.Errorf("ожидался текст 'Мир', получено %q", clean)
	}
}

// TestParseTextURLContext проверяет обработку ссылки внутри произвольного текста.
func TestParseTextURLContext(t *testing.T) {
	base := "😀Привет" // emoji занимает две позиции UTF-16
	add := " переходи в [группу](https://example.com)!"
	ent, clean := parseTextURL(add, utf16Len(base))
	if ent == nil {
		t.Fatalf("сущность не найдена")
	}
	prefixLen := utf16Len(" переходи в ")
	if ent.Offset != utf16Len(base)+prefixLen {
		t.Errorf("ожидался offset %d, получен %d", utf16Len(base)+prefixLen, ent.Offset)
	}
	if ent.Length != utf16Len("группу") {
		t.Errorf("ожидалась длина %d, получена %d", utf16Len("группу"), ent.Length)
	}
	if clean != " переходи в группу!" {
		t.Errorf("ожидался текст ' переходи в группу!', получено %q", clean)
	}
}
