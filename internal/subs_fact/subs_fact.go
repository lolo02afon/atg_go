package subs_fact

import (
	"log"
	"math/rand"
	"time"

	"atg_go/pkg/storage"
	telegramsubs "atg_go/pkg/telegram/subs_active"
)

// rnd нужен для пауз между подписками.
var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

// SyncWithAccountsFact приводит количество подписок в order_account_subs
// в соответствие с полем accounts_number_fact заказа.
func SyncWithAccountsFact(db *storage.DB) error {
	orders, err := db.GetOrdersForMonitoring()
	if err != nil {
		return err
	}
	for _, o := range orders {
		current, err := db.CountOrderSubs(o.ID)
		if err != nil {
			log.Printf("[SUBS_FACT] не удалось получить количество подписок для заказа %d: %v", o.ID, err)
			continue
		}
		if current < o.AccountsNumberFact {
			need := o.AccountsNumberFact - current
			accounts, err := db.GetRandomAccountsForOrder(o.ID, need)
			if err != nil {
				log.Printf("[SUBS_FACT] не удалось выбрать аккаунты для заказа %d: %v", o.ID, err)
				continue
			}
			for _, acc := range accounts {
				if err := telegramsubs.SubscribeAccount(db, acc, o.URLDefault); err != nil {
					log.Printf("[SUBS_FACT] аккаунт %d не смог подписаться на заказ %d: %v", acc.ID, o.ID, err)
					continue
				}
				if err := db.AddOrderAccountSub(o.ID, acc.ID); err != nil {
					log.Printf("[SUBS_FACT] не удалось записать подписку аккаунта %d на заказ %d: %v", acc.ID, o.ID, err)
				}
				time.Sleep(time.Duration(rnd.Intn(3)+2) * time.Second)
			}
		} else if current > o.AccountsNumberFact {
			excess := current - o.AccountsNumberFact
			if err := db.RemoveOrderAccountSubs(o.ID, excess); err != nil {
				log.Printf("[SUBS_FACT] не удалось удалить лишние подписки заказа %d: %v", o.ID, err)
			}
		}
	}
	return nil
}
