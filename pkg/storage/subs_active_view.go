package storage

import (
	"strconv"

	"atg_go/models"
)

// GetAccountsForPostView возвращает авторизованные аккаунты,
// подписанные на канал заказа и ещё не просмотревшие заданный пост.
// channelID и messageID должны быть числовыми идентификаторами канала и поста.
func (db *DB) GetAccountsForPostView(orderID, channelID, messageID int) ([]models.Account, error) {
	chID := strconv.FormatInt(int64(channelID), 10)
	msgID := strconv.FormatInt(int64(messageID), 10)
	condition := `a.is_authorized = true AND a.account_monitoring = false AND a.account_generator_category = false AND EXISTS (
        SELECT 1 FROM order_account_subs oas WHERE oas.account_id = a.id AND oas.order_id = $1
    ) AND NOT EXISTS (
        SELECT 1 FROM activity act WHERE act.id_account = a.id AND act.id_channel = $2 AND act.id_message = $3 AND act.activity_type = $4
    )`
	return db.getAccounts(condition, orderID, chID, msgID, ActivityTypeSubsActiveView)
}
