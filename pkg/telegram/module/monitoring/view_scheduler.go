package monitoring

import (
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	postaction "atg_go/pkg/telegram/post"
	view "atg_go/pkg/telegram/view"
)

// schedulePostViews распределяет просмотры, реакции и репосты поста по времени.
// Реакции и репосты выполняются вместе с просмотром, но не при каждом просмотре.
func schedulePostViews(db *storage.DB, post models.ChannelPost, theory models.ChannelPostTheory, theoryID int) {
	// Определяем ID канала заказа
	order, err := db.GetOrderByID(post.OrderID)
	if err != nil || order.ChannelTGID == nil {
		log.Printf("[MONITORING] не удалось получить канал заказа: %v", err)
		return
	}
	channelID, err := strconv.Atoi(*order.ChannelTGID)
	if err != nil {
		log.Printf("[MONITORING] некорректный channel_tgid %s: %v", *order.ChannelTGID, err)
		return
	}
	// Извлекаем идентификатор поста из ссылки
	parts := strings.Split(strings.TrimSuffix(post.PostURL, "/"), "/")
	msgID, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		log.Printf("[MONITORING] некорректная ссылка на пост: %v", err)
		return
	}

	// Выбираем аккаунты, подписанные на канал заказа и ещё не просмотревшие пост
	accounts, err := db.GetAccountsForPostView(post.OrderID, channelID, msgID)
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

	// Подготавливаем счётчики оставшихся действий для равномерного распределения
	totalViews := 0
	for _, p := range periods {
		totalViews += p.count
	}
	viewsLeft := totalViews
	reactionsLeft := theory.Reaction24HourTheory
	repostsLeft := theory.Repost24HourTheory

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

			// Определяем, нужны ли реакция и репост для данного просмотра
			react, repost := false, false
			if viewsLeft > 0 {
				if reactionsLeft > 0 && rand.Float64() < float64(reactionsLeft)/float64(viewsLeft) {
					react = true
					reactionsLeft--
				}
				if repostsLeft > 0 && rand.Float64() < float64(repostsLeft)/float64(viewsLeft) {
					repost = true
					repostsLeft--
				}
				viewsLeft--
			}

			go func(a models.Account, column string, d time.Duration, doReact, doRepost bool) {
				time.AfterFunc(d, func() {
					if err := view.ViewPost(db, a, post.PostURL); err != nil {
						log.Printf("[MONITORING] просмотр поста не выполнен: %v", err)
						return
					}
					if err := db.IncrementChannelPostFact(theoryID, column); err != nil {
						log.Printf("[MONITORING] обновление факта просмотров: %v", err)
					}
					if doReact {
						if err := postaction.SendReaction(db, a, post.PostURL); err != nil {
							log.Printf("[MONITORING] реакция не выполнена: %v", err)
						} else if err := db.IncrementChannelPostFact(theoryID, "reaction_24hour_fact"); err != nil {
							log.Printf("[MONITORING] обновление факта реакций: %v", err)
						}
					}
					if doRepost {
						if err := postaction.SendRepost(db, a, post.PostURL); err != nil {
							log.Printf("[MONITORING] репост не выполнен: %v", err)
						} else if err := db.IncrementChannelPostFact(theoryID, "repost_24hour_fact"); err != nil {
							log.Printf("[MONITORING] обновление факта репостов: %v", err)
						}
					}
				})
			}(acc, p.column, delay, react, repost)
		}
	}
}
