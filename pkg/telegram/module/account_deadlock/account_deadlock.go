package account_deadlock

import (
	"fmt"
	"sync"
)

var (
	globalMu     sync.Mutex
	accountLocks = make(map[int]*sync.Mutex)
)

// LockAccount пытается захватить мьютекс для указанного аккаунта.
// Если аккаунт уже используется, возвращается ошибка.
func LockAccount(accountID int) error {
	globalMu.Lock()
	lock, ok := accountLocks[accountID]
	if !ok {
		lock = &sync.Mutex{}
		accountLocks[accountID] = lock
	}
	globalMu.Unlock()

	if !lock.TryLock() {
		return fmt.Errorf("аккаунт %d уже используется", accountID)
	}
	return nil
}

// UnlockAccount освобождает мьютекс для указанного аккаунта.
func UnlockAccount(accountID int) {
	globalMu.Lock()
	lock := accountLocks[accountID]
	globalMu.Unlock()
	if lock != nil {
		lock.Unlock()
	}
}
