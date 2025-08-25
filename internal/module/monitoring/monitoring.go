package monitoring

import (
	"atg_go/pkg/storage"
	tgmonitor "atg_go/pkg/telegram/module/monitoring"
)

// Run запускает фоновый мониторинг каналов заказов.
// Работает пока активен сервер.
func Run(db *storage.DB) {
	tgmonitor.Start(db)
}
