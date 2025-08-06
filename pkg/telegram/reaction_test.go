package telegram

import (
	"testing"
)

// TestPickReactionFallback проверяет, что при выборе запрещённой реакции
// возвращается первая разрешённая.
func TestPickReactionFallback(t *testing.T) {
	rnd.Seed(1)
	allowed := []string{"❤️"}
	base := []string{"👍"}
	r := pickReaction(base, allowed)
	if r != "❤️" {
		t.Fatalf("ожидалась реакция ❤️, получено %s", r)
	}
}

// TestPickReactionAllowed убеждается, что разрешённая реакция может быть выбрана.
func TestPickReactionAllowed(t *testing.T) {
	rnd.Seed(2)
	allowed := []string{"❤️", "👍"}
	base := []string{"❤️", "👍"}
	r := pickReaction(base, allowed)
	if r != "👍" && r != "❤️" {
		t.Fatalf("неожиданная реакция: %s", r)
	}
	// Дополнительно проверяем, что выбранная реакция входит в список allowed.
	ok := false
	for _, a := range allowed {
		if r == a {
			ok = true
		}
	}
	if !ok {
		t.Fatalf("выбрана реакция, не разрешённая администраторами: %s", r)
	}
}
