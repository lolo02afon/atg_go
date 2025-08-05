package telegram

import (
	"testing"

	"github.com/gotd/td/tg"
)

// TestSelectTargetMessage_NewestMessage проверяет, что выбирается самое новое сообщение без реакций.
func TestSelectTargetMessage_NewestMessage(t *testing.T) {
	msgs := []*tg.Message{{ID: 3}, {ID: 2}, {ID: 1}}
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

// TestSelectTargetMessage_SkipReactions убеждается, что сообщения с реакциями пропускаются.
func TestSelectTargetMessage_SkipReactions(t *testing.T) {
	msgs := []*tg.Message{
		{ID: 3, Reactions: tg.MessageReactions{Results: []tg.ReactionCount{{Reaction: &tg.ReactionEmoji{Emoticon: "❤️"}, Count: 1}}}},
		{ID: 2},
		{ID: 1},
	}
	msg, err := selectTargetMessage(msgs)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if msg.ID != 2 {
		t.Fatalf("ожидался ID 2, получено %d", msg.ID)
	}
}

// TestSelectTargetMessage_AllWithReactions проверяет, что возвращается ошибка, если все сообщения уже с реакциями.
func TestSelectTargetMessage_AllWithReactions(t *testing.T) {
	msgs := []*tg.Message{
		{ID: 2, Reactions: tg.MessageReactions{Results: []tg.ReactionCount{{Reaction: &tg.ReactionEmoji{Emoticon: "❤️"}, Count: 1}}}},
		{ID: 1, Reactions: tg.MessageReactions{Results: []tg.ReactionCount{{Reaction: &tg.ReactionEmoji{Emoticon: "😂"}, Count: 1}}}},
	}
	if _, err := selectTargetMessage(msgs); err == nil {
		t.Fatalf("ожидалась ошибка, но её нет")
	}
}
