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

	"github.com/gotd/td/tg"
)

// orderInfo хранит сведения о заказе, необходимые для расчёта метрик.
type orderInfo struct {
	id  int
	url string
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

// Connect присоединяет модуль мониторинга к существующему клиенту Telegram.
// Предполагается, что клиент и диспетчер уже инициализированы и работают.
func Connect(ctx context.Context, api *tg.Client, dispatcher *tg.UpdateDispatcher, db *storage.DB, accountID int) {
	orders, err := db.GetOrdersForMonitoring()
	if err != nil {
		log.Printf("[MONITORING] получение заказов: %v", err)
		return
	}

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

			// Считаем фактическое число подписанных аккаунтов для заказа
			view, err := db.CountOrderSubs(o.id)
			if err != nil {
				log.Printf("[MONITORING] подсчёт подписчиков заказа %d: %v", o.id, err)
			}

			// Реакции: от 0.5% до 2% от фактического числа просмотров
			reaction := randomByPercent(view, 0.5, 2)
			// Репосты: от 2% до 10% от фактического числа просмотров
			repost := randomByPercent(view, 2, 10)

			// Указатели устанавливаются только при наличии активной аудитории
			var viewPtr, reactionPtr, repostPtr *int
			if view > 0 {
				viewPtr = &view
				reactionPtr = &reaction
				repostPtr = &repost
			}

			cp := models.ChannelPost{
				OrderID:            o.id,
				PostDateTime:       postTime,
				PostURL:            link,
				SubsActiveView:     viewPtr,
				SubsActiveReaction: reactionPtr,
				SubsActiveRepost:   repostPtr,
			}
			// Сохраняем пост и получаем его идентификатор для последующей теории
			postID, err := db.CreateChannelPost(cp)
			if err != nil {
				log.Printf("[MONITORING] сохранение поста: %v", err)
			} else {
				// Формируем прогноз просмотров по группам часов
				// и ограничиваем суммарное значение фактическим максимумом
				view1 := randomByPercent(view, 20.6, 25.7)
				remain := view - view1
				view23 := randomByPercent(view, 17.2, 21.7)
				if view23 > remain {
					view23 = remain
				}
				remain -= view23
				view46 := randomByPercent(view, 14.9, 19.4)
				if view46 > remain {
					view46 = remain
				}
				remain -= view46
				view724 := randomByPercent(view, 31.9, 39.4)
				if view724 > remain {
					view724 = remain
				}
				theory := models.ChannelPostTheory{
					ChannelPostID:        postID,
					View1HourTheory:      float64(view1),
					View23HourTheory:     float64(view23),
					View46HourTheory:     float64(view46),
					View724HourTheory:    float64(view724),
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

	// Подписываемся на каналы заказов и включаем уведомления
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
		if err := base.Modf_JoinChannel(ctx, api, ch, db, accountID); err != nil && !strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
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
		orderMap[ch.ID] = orderInfo{id: o.ID, url: o.URLDefault}
	}
}
