package account_deadlock

import pkgdeadlock "atg_go/pkg/telegram/module/account_deadlock"

// LockAccount экспортирует блокировку аккаунта для внутренних пакетов.
func LockAccount(accountID int) error {
	return pkgdeadlock.LockAccount(accountID)
}

// UnlockAccount освобождает блокировку аккаунта.
func UnlockAccount(accountID int) {
	pkgdeadlock.UnlockAccount(accountID)
}
