package channel_duplicate

import (
	"testing"

	"github.com/gotd/td/tg"
)

// TestAdjustEntitiesAfterRemoval проверяет корректное смещение форматирования после удаления текста.
func TestAdjustEntitiesAfterRemoval(t *testing.T) {
	text := "Удалить: жирный текст"
	remove := "Удалить: "
	bold := &tg.MessageEntityBold{
		Offset: utf16Len("Удалить: "),
		Length: utf16Len("жирный"),
	}
	ents := []tg.MessageEntityClass{bold}
	adjusted := adjustEntitiesAfterRemoval(ents, text, remove)
	if len(adjusted) != 1 {
		t.Fatalf("ожидалось 1 сущность, получено %d", len(adjusted))
	}
	b, ok := adjusted[0].(*tg.MessageEntityBold)
	if !ok {
		t.Fatalf("ожидался тип MessageEntityBold")
	}
	if b.Offset != 0 {
		t.Fatalf("ожидался offset 0, получено %d", b.Offset)
	}
	if b.Length != utf16Len("жирный") {
		t.Fatalf("ожидалась длина %d, получено %d", utf16Len("жирный"), b.Length)
	}
}
