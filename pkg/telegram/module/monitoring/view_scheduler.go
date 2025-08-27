package monitoring

import (
	"log"
	"math/rand"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	view "atg_go/pkg/telegram/view"
)

// schedulePostViews равномерно распределяет просмотры поста по временным промежуткам.
// Для каждого интервала создаётся фоновая задача, которая вызывает просмотр поста
// выбранным аккаунтом и увеличивает соответствующее поле факта.
func schedulePostViews(db *storage.DB, post models.ChannelPost, theory models.ChannelPostTheory, theoryID int) {
	accounts, err := db.GetAuthorizedAccounts()
	if err != nil || len(accounts) == 0 {
		log.Printf("[MONITORING] не удалось получить аккаунты для просмотров: %v", err)
		return
	}

	type period struct {
		count  int
		start  time.Duration
		end    time.Duration
		column string
	}

	periods := []period{
		{int(theory.View1HourTheory), 0, time.Hour, "view_1hour_fact"},
		{int(theory.View23HourTheory), 2 * time.Hour, 3 * time.Hour, "view_2_3hour_fact"},
		{int(theory.View46HourTheory), 4 * time.Hour, 6 * time.Hour, "view_4_6hour_fact"},
		{int(theory.View724HourTheory), 7 * time.Hour, 24 * time.Hour, "view_7_24hour_fact"},
	}

	for _, p := range periods {
		if p.count <= 0 {
			continue
		}
		step := (p.end - p.start) / time.Duration(p.count)
		for i := 0; i < p.count; i++ {
			acc := accounts[rand.Intn(len(accounts))]
			delay := post.PostDateTime.Add(p.start + step*time.Duration(i)).Sub(time.Now())
			if delay < 0 {
				delay = 0
			}
			go func(a models.Account, column string, d time.Duration) {
				time.AfterFunc(d, func() {
					if err := view.ViewPost(db, a, post.PostURL); err != nil {
						log.Printf("[MONITORING] просмотр поста не выполнен: %v", err)
						return
					}
					if err := db.IncrementChannelPostFact(theoryID, column); err != nil {
						log.Printf("[MONITORING] обновление факта просмотров: %v", err)
					}
				})
			}(acc, p.column, delay)
		}
	}
}
