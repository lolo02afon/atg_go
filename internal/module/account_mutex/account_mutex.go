package account_mutex

import pkgmutex "atg_go/pkg/telegram/module/account_mutex"

// LockAccount экспортирует блокировку аккаунта для внутренних пакетов.
func LockAccount(accountID int) error {
	return pkgmutex.LockAccount(accountID)
}

// UnlockAccount освобождает блокировку аккаунта.
func UnlockAccount(accountID int) {
	pkgmutex.UnlockAccount(accountID)
}
