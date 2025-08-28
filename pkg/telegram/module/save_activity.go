package module

import "atg_go/pkg/storage"

// Функции ниже сохраняют поддерживаемые типы активности. Отказ от
// универсального метода заставляет явно добавлять новый тип активности,
// что уменьшает риск нелегитимных значений в таблице.

// SaveReactionActivity сохраняет реакцию, что позволяет не передавать
// строковый тип действия из вызывающего кода и тем самым избегать опечаток.
func SaveReactionActivity(db *storage.DB, accountID, channelID, messageID int) error {
	return db.SaveReaction(accountID, channelID, messageID)
}

// SaveCommentActivity сохраняет комментарий по той же причине: тип действия
// фиксирован и не зависит от внешнего ввода.
// messageID — ID поста, к которому оставлен комментарий.
func SaveCommentActivity(db *storage.DB, accountID, channelID, messageID int) error {
	return db.SaveComment(accountID, channelID, messageID)
}

// SaveViewActivity фиксирует просмотр поста активной аудиторией.
// messageID — идентификатор поста канала, открытого для увеличения просмотров.
func SaveViewActivity(db *storage.DB, accountID, channelID, messageID int) error {
	return db.SaveSubsActiveView(accountID, channelID, messageID)
}
