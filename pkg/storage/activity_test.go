package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
)

// dummyDriver предоставляет минимальную реализацию драйвера SQL
// для перехвата выполняемых запросов без реальной БД.
type dummyDriver struct{}

type dummyConn struct{}

type dummyResult struct{}

// executedQueries хранит все запросы Exec, чтобы проверять их содержимое
var executedQueries []string

func (d *dummyDriver) Open(name string) (driver.Conn, error) { return &dummyConn{}, nil }

func (c *dummyConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (c *dummyConn) Close() error              { return nil }
func (c *dummyConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }

// ExecContext сохраняет текст запроса и всегда успешно завершается
func (c *dummyConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	executedQueries = append(executedQueries, query)
	return dummyResult{}, nil
}

func (c *dummyConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return nil, errors.New("not implemented")
}

func (dummyResult) LastInsertId() (int64, error) { return 0, nil }
func (dummyResult) RowsAffected() (int64, error) { return 1, nil }

func init() {
	sql.Register("dummy", &dummyDriver{})
}

// TestSaveActivityDuplicate проверяет, что повторная вставка одинаковых
// активностей не вызывает ошибку и запрос содержит ON CONFLICT DO NOTHING.
func TestSaveActivityDuplicate(t *testing.T) {
	executedQueries = nil
	db, err := sql.Open("dummy", "")
	if err != nil {
		t.Fatalf("не удалось открыть фейковую БД: %v", err)
	}
	storageDB := &DB{Conn: db}

	if err := storageDB.SaveActivity(1, 2, 3, ActivityTypeComment); err != nil {
		t.Fatalf("первая вставка завершилась ошибкой: %v", err)
	}
	if err := storageDB.SaveActivity(1, 2, 3, ActivityTypeComment); err != nil {
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
