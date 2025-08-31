package channel_duplicate

import "testing"

// TestParseTextURL проверяет корректность вычисления смещений в UTF-16.
func TestParseTextURL(t *testing.T) {
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
