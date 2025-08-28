package monitoring

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"atg_go/models"
	"atg_go/pkg/storage"
	base "atg_go/pkg/telegram/module"
	accountmutex "atg_go/pkg/telegram/module/account_mutex"

	"github.com/gotd/td/tg"
)

type orderInfo struct {
	id                 int
	url                string
	accountsNumberFact int
}

// randomByPercent возвращает число, равное случайному проценту от base.
// Диапазон процентов задаётся в min и max, округление вверх или вниз выбирается случайно.
func randomByPercent(base int, min, max float64) int {
	if base == 0 {
		return 0
	}
	percent := min + rand.Float64()*(max-min)
	value := float64(base) * percent / 100
	floor := int(value)
	if value == float64(floor) {
		return floor
	}
	if rand.Intn(2) == 0 {
		return floor
	}
	return floor + 1
}

// Start запускает отслеживание новых постов на каналах заказов.
// Используется первым доступным мониторинг-аккаунтом.
func Start(db *storage.DB) {
	go func() {
		if err := run(db); err != nil {
			log.Printf("[MONITORING] остановлено: %v", err)
		}
	}()
}

// run выполняет инициализацию клиента Telegram и обрабатывает обновления.
func run(db *storage.DB) error {
	rand.Seed(time.Now().UnixNano())
	accounts, err := db.GetMonitoringAccounts()
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		return fmt.Errorf("нет аккаунтов для мониторинга")
	}
	acc := accounts[0]

	if err := accountmutex.LockAccount(acc.ID); err != nil {
		return err
	}
	defer accountmutex.UnlockAccount(acc.ID)

	orders, err := db.GetOrdersForMonitoring()
	if err != nil {
		return err
	}

	dispatcher := tg.NewUpdateDispatcher()
	orderMap := make(map[int64]orderInfo)

	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, upd *tg.UpdateNewChannelMessage) error {
		msg, ok := upd.Message.(*tg.Message)
		if !ok {
			return nil
		}
		peer, ok := msg.PeerID.(*tg.PeerChannel)
		if !ok {
			return nil
		}
		if o, ok := orderMap[peer.ChannelID]; ok {
			postTime := time.Unix(int64(msg.Date), 0)
			link := strings.TrimSuffix(o.url, "/") + "/" + strconv.Itoa(msg.ID)

			// Берём целевое число просмотров из фактического количества аккаунтов заказа
			view := o.accountsNumberFact

			// Реакции: от 0.5% до 2% от целевого числа просмотров
			reaction := randomByPercent(view, 0.5, 2)
			// Репосты: от 2% до 10% от целевого числа просмотров
			repost := randomByPercent(view, 2, 10)

			cp := models.ChannelPost{
				OrderID:            o.id,
				PostDateTime:       postTime,
				PostURL:            link,
				SubsActiveView:     &view,
				SubsActiveReaction: &reaction,
				SubsActiveRepost:   &repost,
			}
			// Сохраняем пост и получаем его идентификатор для последующей теории
			postID, err := db.CreateChannelPost(cp)
			if err != nil {
				log.Printf("[MONITORING] сохранение поста: %v", err)
			} else {
				// Формируем прогноз просмотров по группам часов
				theory := models.ChannelPostTheory{
					ChannelPostID:        postID,
					View1HourTheory:      float64(randomByPercent(view, 20.6, 25.7)),
					View23HourTheory:     float64(randomByPercent(view, 6.7, 11.0)),
					View46HourTheory:     float64(randomByPercent(view, 3.7, 6.3)),
					View724HourTheory:    float64(randomByPercent(view, 0.5, 3.2)),
					Reaction24HourTheory: reaction,
					Repost24HourTheory:   repost,
				}
				// Создаём прогноз и получаем его идентификатор
				theoryID, err := db.CreateChannelPostTheory(theory)
				if err != nil {
					log.Printf("[MONITORING] сохранение теории просмотров: %v", err)
				} else {
					// Создаём запись фактических просмотров с нулевыми значениями
					fact := models.ChannelPostFact{ChannelPostTheoryID: theoryID}
					if err := db.CreateChannelPostFact(fact); err != nil {
						log.Printf("[MONITORING] сохранение факта просмотров: %v", err)
					} else {
						// Планируем просмотры поста по интервалам
						schedulePostViews(db, cp, theory, theoryID)
					}
				}
			}
		}
		return nil
	})

	client, err := base.Modf_AccountInitialization(acc.ApiID, acc.ApiHash, acc.Phone, acc.Proxy, nil, db.Conn, acc.ID, dispatcher)
	if err != nil {
		return err
	}

	ctx := context.Background()

	return client.Run(ctx, func(ctx context.Context) error {
		api := tg.NewClient(client)

		// Подписываемся на каналы и включаем уведомления
		for _, o := range orders {
			username, err := base.Modf_ExtractUsername(o.URLDefault)
			if err != nil {
				log.Printf("[MONITORING] некорректная ссылка %s: %v", o.URLDefault, err)
				continue
			}
			resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
			if err != nil {
				log.Printf("[MONITORING] не удалось получить канал %s: %v", o.URLDefault, err)
				continue
			}
			ch, err := base.Modf_FindChannel(resolved.GetChats())
			if err != nil {
				log.Printf("[MONITORING] канал %s не найден: %v", o.URLDefault, err)
				continue
			}
			if err := base.Modf_JoinChannel(ctx, api, ch, db, acc.ID); err != nil && !strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
				log.Printf("[MONITORING] подписка на %s: %v", o.URLDefault, err)
			}
			settings := tg.InputPeerNotifySettings{}
			settings.SetMuteUntil(0)
			_, err = api.AccountUpdateNotifySettings(ctx, &tg.AccountUpdateNotifySettingsRequest{
				Peer:     &tg.InputNotifyPeer{Peer: &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}},
				Settings: settings,
			})
			if err != nil {
				log.Printf("[MONITORING] уведомления %s: %v", o.URLDefault, err)
			}
			if o.ChannelTGID == nil {
				_ = db.SetOrderChannelTGID(o.ID, fmt.Sprintf("%d", ch.ID))
			}
			orderMap[ch.ID] = orderInfo{id: o.ID, url: o.URLDefault, accountsNumberFact: o.AccountsNumberFact}
		}

		// держим соединение активным, пока контекст не будет отменён
		<-ctx.Done()
		return nil
	})
}
