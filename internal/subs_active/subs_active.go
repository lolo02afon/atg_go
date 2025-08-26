package subs_active

import (
	"log"
	"math/rand"
	"time"

	"atg_go/pkg/storage"
	telegramsubs "atg_go/pkg/telegram/subs_active"
)

// rnd нужен для пауз между подписками.
var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

// ActivateSubscriptions проверяет заказы и подписывает недостающие аккаунты на их каналы.
func ActivateSubscriptions(db *storage.DB) error {
	orders, err := db.GetOrdersForMonitoring()
	if err != nil {
		return err
	}
	for _, o := range orders {
		if o.SubsActiveCount == nil || *o.SubsActiveCount <= 0 || o.URLDefault == "" {
			continue
		}
		current, err := db.CountOrderSubs(o.ID)
		if err != nil {
			log.Printf("[SUBS_ACTIVE] не удалось получить количество подписок для заказа %d: %v", o.ID, err)
			continue
		}
		need := *o.SubsActiveCount - current
		if need <= 0 {
			continue
		}
		accounts, err := db.GetRandomAccountsForOrder(o.ID, need)
		if err != nil {
			log.Printf("[SUBS_ACTIVE] не удалось выбрать аккаунты для заказа %d: %v", o.ID, err)
			continue
		}
		for _, acc := range accounts {
			if err := telegramsubs.SubscribeAccount(db, acc, o.URLDefault); err != nil {
				log.Printf("[SUBS_ACTIVE] аккаунт %d не смог подписаться на заказ %d: %v", acc.ID, o.ID, err)
				continue
			}
			if err := db.AddOrderAccountSub(o.ID, acc.ID); err != nil {
				log.Printf("[SUBS_ACTIVE] не удалось записать подписку аккаунта %d на заказ %d: %v", acc.ID, o.ID, err)
			}
			time.Sleep(time.Duration(rnd.Intn(3)+2) * time.Second)
		}
	}
	return nil
}
