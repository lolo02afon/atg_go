package account_mutex

import (
	"fmt"
	"log"
	"sync"
)

var (
	globalMu       sync.Mutex
	accountLocks   = make(map[int]*sync.Mutex)
	lockedAccounts = make(map[int]struct{})
)

// lockedIDs возвращает список текущих заблокированных аккаунтов.
// Предполагается, что глобальный мьютекс уже захвачен.
func lockedIDs() []int {
	ids := make([]int, 0, len(lockedAccounts))
	for id := range lockedAccounts {
		ids = append(ids, id)
	}
	return ids
}

// LockAccount пытается захватить мьютекс для указанного аккаунта.
// Если аккаунт уже используется, возвращается ошибка. Дополнительно
// выводится список заблокированных аккаунтов для упрощения диагностики.
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
		globalMu.Lock()
		current := lockedIDs()
		globalMu.Unlock()
		log.Printf("[MUTEX] аккаунт %d занят; заблокированы: %v", accountID, current)
		return fmt.Errorf("аккаунт %d уже используется", accountID)
	}

	globalMu.Lock()
	lockedAccounts[accountID] = struct{}{}
	current := lockedIDs()
	globalMu.Unlock()

	log.Printf("[MUTEX] аккаунт %d заблокирован; заблокированы: %v", accountID, current)
	return nil
}

// UnlockAccount освобождает мьютекс для указанного аккаунта
// и выводит актуальный список заблокированных аккаунтов.
func UnlockAccount(accountID int) {
	globalMu.Lock()
	lock := accountLocks[accountID]
	delete(lockedAccounts, accountID)
	current := lockedIDs()
	globalMu.Unlock()
	if lock != nil {
		lock.Unlock()
		log.Printf("[MUTEX] аккаунт %d разблокирован; заблокированы: %v", accountID, current)
	}
}
