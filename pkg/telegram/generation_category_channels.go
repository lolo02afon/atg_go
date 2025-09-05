package telegram

import (
	"atg_go/models"
	"atg_go/pkg/storage"
	gcc "atg_go/pkg/telegram/generation_category_channels"
)

// GetChannelRecommendations делегирует получение похожих каналов подпакету generation_category_channels.
func GetChannelRecommendations(db *storage.DB, acc models.Account, channelURL string) ([]string, error) {
	return gcc.GetChannelRecommendations(db, acc, channelURL)
}
