package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"
)

// monitoringTestDriver предоставляет упрощённый драйвер БД для тестирования GetOrdersForMonitoring.
type monitoringTestDriver struct{}

type monitoringTestConn struct{}

type monitoringTestRows struct {
	columns []string
	data    [][]driver.Value
	idx     int
}

type monitoringTestTx struct{}

type monitoringDummyResult struct{}

func (monitoringTestDriver) Open(name string) (driver.Conn, error) { return &monitoringTestConn{}, nil }

func (c *monitoringTestConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (c *monitoringTestConn) Close() error              { return nil }
func (c *monitoringTestConn) Begin() (driver.Tx, error) { return &monitoringTestTx{}, nil }

func (c *monitoringTestConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return &monitoringTestRows{
		columns: []string{"id", "url_default", "channel_tgid", "accounts_number_fact", "subs_active_count"},
		data: [][]driver.Value{
			{int64(1), "https://t.me/c/123/abc", "123", int64(10), int64(5)},
			{int64(2), "https://t.me/c/456/def", "456", int64(20), nil},
		},
	}, nil
}

func (c *monitoringTestConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return monitoringDummyResult{}, nil
}

func (monitoringTestTx) Commit() error   { return nil }
func (monitoringTestTx) Rollback() error { return nil }

func (r *monitoringTestRows) Columns() []string { return r.columns }
func (r *monitoringTestRows) Close() error      { return nil }
func (r *monitoringTestRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.idx])
	r.idx++
	return nil
}

func (monitoringDummyResult) LastInsertId() (int64, error) { return 0, nil }
func (monitoringDummyResult) RowsAffected() (int64, error) { return 0, nil }

func init() { sql.Register("monitorDummy", monitoringTestDriver{}) }

// TestGetOrdersForMonitoringSubsActiveCount проверяет корректное чтение subs_active_count.
func TestGetOrdersForMonitoringSubsActiveCount(t *testing.T) {
	db, err := sql.Open("monitorDummy", "")
	if err != nil {
		t.Fatalf("не удалось открыть мок БД: %v", err)
	}
	defer func() { _ = db.Close() }()
	storageDB := &DB{Conn: db}

	orders, err := storageDB.GetOrdersForMonitoring()
	if err != nil {
		t.Fatalf("запрос заказов завершился ошибкой: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("ожидалось 2 заказа, получено %d", len(orders))
	}
	if orders[0].SubsActiveCount == nil || *orders[0].SubsActiveCount != 5 {
		t.Fatalf("ожидалось SubsActiveCount = 5, получено %v", orders[0].SubsActiveCount)
	}
	if orders[1].SubsActiveCount != nil {
		t.Fatalf("ожидалось SubsActiveCount = nil, получено %v", orders[1].SubsActiveCount)
	}
}
