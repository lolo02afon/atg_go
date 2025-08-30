package channel_duplicate

import (
	"testing"

	"github.com/gotd/td/tg"
)

// TestIsAdvertisement проверяет определение рекламных постов.
func TestIsAdvertisement(t *testing.T) {
	tests := []struct {
		name string
		msg  *tg.Message
		want bool
	}{
		{
			name: "хештег",
			msg:  &tg.Message{Message: "текст #реклама"},
			want: true,
		},
		{
			name: "utm_medium",
			msg:  &tg.Message{Message: "https://example.com?utm_medium=promo"},
			want: true,
		},
		{
			name: "erid в ссылке",
			msg: &tg.Message{Message: "смотри", Entities: []tg.MessageEntityClass{
				&tg.MessageEntityTextURL{URL: "https://a.com?erid=123"},
			}},
			want: true,
		},
		{
			name: "чистый",
			msg:  &tg.Message{Message: "привет мир"},
			want: false,
		},
	}

	for _, tt := range tests {
		if got := isAdvertisement(tt.msg); got != tt.want {
			t.Errorf("%s: ожидалось %v, получено %v", tt.name, tt.want, got)
		}
	}
}
