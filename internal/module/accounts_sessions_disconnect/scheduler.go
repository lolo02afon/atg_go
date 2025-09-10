package accounts_sessions_disconnect

import (
	"log"
	"time"

	"atg_go/pkg/storage"
	telegrammodule "atg_go/pkg/telegram/module/accounts_sessions_disconnect"
)

// startBackgroundDisconnect запускает бесконечный цикл, который
// выполняет отключение сессий каждый день в 02:00 и 11:00 по МСК.
func startBackgroundDisconnect(db *storage.DB) {
	go func() {
		for {
			now := time.Now()
			// Рассчитываем ближайшее целевое время.
			year, month, day := now.Date()
			loc := now.Location()
			t02 := time.Date(year, month, day, 2, 0, 0, 0, loc)
			t11 := time.Date(year, month, day, 11, 0, 0, 0, loc)
			var next time.Time
			switch {
			case now.Before(t02):
				next = t02
			case now.Before(t11):
				next = t11
			default:
				next = t02.AddDate(0, 0, 1)
			}
			time.Sleep(next.Sub(now))
			if _, err := telegrammodule.DisconnectSuspiciousSessions(db, 0, 0); err != nil {
				// Фиксируем ошибку, чтобы отслеживать проблемы
				log.Printf("[ACCOUNTS SESSIONS DISCONNECT] ошибка фонового отключения: %v", err)
			}
		}
	}()
}
