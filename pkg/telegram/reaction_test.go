package telegram

import (
	"testing"

	"github.com/gotd/td/tg"
)

// TestSelectTargetMessage_LastMessage проверяет, что выбирается последнее сообщение.
func TestSelectTargetMessage_LastMessage(t *testing.T) {
	msgs := []*tg.Message{{ID: 1}, {ID: 2}, {ID: 3}}
	msg, err := selectTargetMessage(msgs)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if msg.ID != 3 {
		t.Fatalf("ожидался ID 3, получено %d", msg.ID)
	}
}

// TestSelectTargetMessage_Empty проверяет, что при отсутствии сообщений возвращается ошибка.
func TestSelectTargetMessage_Empty(t *testing.T) {
	if _, err := selectTargetMessage([]*tg.Message{}); err == nil {
		t.Fatalf("ожидалась ошибка, но её нет")
	}
}

// TestSelectTargetMessage_IgnoresReactions убеждается, что наличие реакций не влияет на выбор.
func TestSelectTargetMessage_IgnoresReactions(t *testing.T) {
	msgs := []*tg.Message{
		{ID: 1},
		{ID: 2, Reactions: &tg.MessageReactions{Results: []tg.ReactionCount{{Reaction: &tg.ReactionEmoji{Emoticon: "❤️"}, Count: 1}}}},
	}
	msg, err := selectTargetMessage(msgs)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if msg.ID != 2 {
		t.Fatalf("ожидался ID 2, получено %d", msg.ID)
	}
}
