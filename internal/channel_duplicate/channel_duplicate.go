package channel_duplicate

import (
	"atg_go/pkg/storage"
	tgdup "atg_go/pkg/telegram/channel_duplicate"
)

// Run запускает фоновую пересылку постов с донорских каналов.
// Работает, пока активен сервер.
func Run(db *storage.DB) {
	tgdup.Start(db)
}
