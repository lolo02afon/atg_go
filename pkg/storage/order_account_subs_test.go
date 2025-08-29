package storage

import (
	"database/sql"
	"strings"
	"testing"
)

// TestAddOrderAccountSubDuplicate проверяет, что повторная вставка одинаковых
// подписок не вызывает ошибку и запрос содержит ON CONFLICT DO NOTHING.
func TestAddOrderAccountSubDuplicate(t *testing.T) {
	executedQueries = nil
	db, err := sql.Open("dummy", "")
	if err != nil {
		t.Fatalf("не удалось открыть фейковую БД: %v", err)
	}
	storageDB := &DB{Conn: db}

	if err := storageDB.AddOrderAccountSub(1, 2); err != nil {
		t.Fatalf("первая вставка завершилась ошибкой: %v", err)
	}
	if err := storageDB.AddOrderAccountSub(1, 2); err != nil {
		t.Fatalf("повторная вставка завершилась ошибкой: %v", err)
	}
	if len(executedQueries) != 2 {
		t.Fatalf("ожидалось 2 запроса, получено %d", len(executedQueries))
	}
	for _, q := range executedQueries {
		if !strings.Contains(q, "ON CONFLICT DO NOTHING") {
			t.Fatalf("в запросе отсутствует ON CONFLICT DO NOTHING: %s", q)
		}
	}
}
