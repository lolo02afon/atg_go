package channel_duplicate

import (
	"testing"

	"github.com/gotd/td/tg"
)

// TestShouldSkip проверяет работу условий пропуска постов.
func TestShouldSkip(t *testing.T) {
	tests := []struct {
		name string
		ps   postSkip
		msg  *tg.Message
		want bool
	}{
		{
			name: "по тексту",
			ps:   postSkip{Text: []string{"текст1"}},
			msg:  &tg.Message{Message: "в посте есть текст1"},
			want: true,
		},
		{
			name: "по ссылке",
			ps:   postSkip{URL: []string{"feroe3"}},
			msg:  &tg.Message{Message: "https://example.com/feroe3"},
			want: true,
		},
		{
			name: "по ссылке в сущности",
			ps:   postSkip{URL: []string{"lolofilms"}},
			msg: &tg.Message{Message: "смотри", Entities: []tg.MessageEntityClass{
				&tg.MessageEntityTextURL{URL: "http://a.com/loloFilms"},
			}},
			want: true,
		},
		{
			name: "нет совпадений",
			ps:   postSkip{Text: []string{"a"}, URL: []string{"b"}},
			msg:  &tg.Message{Message: "привет"},
			want: false,
		},
	}

	for _, tt := range tests {
		if got := shouldSkip(tt.msg, tt.ps); got != tt.want {
			t.Errorf("%s: ожидалось %v, получено %v", tt.name, tt.want, got)
		}
	}
}
