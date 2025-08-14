package account_mutex

import (
	"fmt"
	"log"
	"sync"
)

var (
	globalMu     sync.Mutex
	accountLocks = make(map[int]*sync.Mutex)
)

// LockAccount пытается захватить мьютекс для указанного аккаунта.
// Если аккаунт уже используется, возвращается ошибка.
// В журнал записывается информация о попытке, успешной блокировке
// и отказе в случае занятости.
func LockAccount(accountID int) error {
	log.Printf("[MUTEX] попытка блокировки аккаунта %d", accountID)

	globalMu.Lock()
	lock, ok := accountLocks[accountID]
	if !ok {
		lock = &sync.Mutex{}
		accountLocks[accountID] = lock
	}
	globalMu.Unlock()

	if !lock.TryLock() {
		log.Printf("[MUTEX] аккаунт %d занят", accountID)
		return fmt.Errorf("аккаунт %d уже используется", accountID)
	}

	log.Printf("[MUTEX] аккаунт %d заблокирован", accountID)
	return nil
}

// UnlockAccount освобождает мьютекс для указанного аккаунта
// и записывает это событие в журнал.
func UnlockAccount(accountID int) {
	globalMu.Lock()
	lock := accountLocks[accountID]
	globalMu.Unlock()
	if lock != nil {
		lock.Unlock()
		log.Printf("[MUTEX] аккаунт %d разблокирован", accountID)
	}
}
