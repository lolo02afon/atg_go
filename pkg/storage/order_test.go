package storage

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"

	"atg_go/models"
)

// orderTestDriver реализует минимальный SQL-драйвер для теста CreateOrder.
// Он возвращает предопределённые ответы и не требует внешних зависимостей.
type orderTestDriver struct{}

type orderTestConn struct{ step int }

type orderTestTx struct{}

type orderTestRows struct {
	columns []string
	data    [][]driver.Value
	idx     int
}

type orderDummyResult struct{}

func (orderTestDriver) Open(name string) (driver.Conn, error) { return &orderTestConn{}, nil }

func (c *orderTestConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (c *orderTestConn) Close() error              { return nil }
func (c *orderTestConn) Begin() (driver.Tx, error) { return &orderTestTx{}, nil }

func (c *orderTestConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	switch c.step {
	case 0:
		// Выборка существующих категорий — возвращаем одну найденную
		c.step++
		return &orderTestRows{columns: []string{"name"}, data: [][]driver.Value{{"известная"}}}, nil
	case 1:
		// Вставка заказа — возвращаем созданную запись
		c.step++
		return &orderTestRows{
			columns: []string{"id", "accounts_number_fact", "date_time", "subs_active_count"},
			data:    [][]driver.Value{{int64(1), int64(0), time.Now(), nil}},
		}, nil
	case 2:
		// Запрос аккаунтов — возвращаем пустой набор
		c.step++
		return &orderTestRows{columns: []string{"id"}, data: [][]driver.Value{}}, nil
	default:
		return nil, errors.New("unexpected query")
	}
}

func (c *orderTestConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	// В тесте не выполняются запросы Exec, но метод должен существовать
	return orderDummyResult{}, nil
}

func (orderTestTx) Commit() error   { return nil }
func (orderTestTx) Rollback() error { return nil }

func (r *orderTestRows) Columns() []string { return r.columns }
func (r *orderTestRows) Close() error      { return nil }
func (r *orderTestRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.idx])
	r.idx++
	return nil
}

func (orderDummyResult) LastInsertId() (int64, error) { return 0, nil }
func (orderDummyResult) RowsAffected() (int64, error) { return 1, nil }

func init() {
	sql.Register("orderDummy", orderTestDriver{})
}

// TestCreateOrderIgnoresUnknownCategories проверяет, что заказ создаётся даже при наличии
// неизвестных категорий, которые логируются и не попадают в итоговый список.
func TestCreateOrderIgnoresUnknownCategories(t *testing.T) {
	db, err := sql.Open("orderDummy", "")
	if err != nil {
		t.Fatalf("не удалось открыть мок БД: %v", err)
	}
	defer func() { _ = db.Close() }()
	storageDB := &DB{Conn: db}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(log.LstdFlags)
	}()

	o := models.Order{
		Name:                 "тест",
		Category:             pq.StringArray{"известная", "неизвестная"},
		URLDescription:       "desc",
		URLDefault:           "https://t.me/c/123/abc",
		AccountsNumberTheory: 0,
		Gender:               pq.StringArray{"male"},
	}
	created, err := storageDB.CreateOrder(o)
	if err != nil {
		t.Fatalf("создание заказа завершилось ошибкой: %v", err)
	}
	if len(created.Category) != 1 || created.Category[0] != "известная" {
		t.Fatalf("категории отфильтрованы неверно: %v", created.Category)
	}
	if !strings.Contains(buf.String(), "категория \"неизвестная\" не найдена") {
		t.Fatalf("в логах отсутствует предупреждение о неизвестной категории: %s", buf.String())
	}
}
