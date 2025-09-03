package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"
)

type commentTestDriver struct{}

type commentTestConn struct{ step int }

type commentTestRows struct {
	columns []string
	data    [][]driver.Value
	idx     int
}

type commentDummyResult struct{}

// commentBanned имитирует содержимое таблицы category_channels_delete.
// Заполняется в тестах перед обращением к базе.
var commentBanned []string

func (commentTestDriver) Open(name string) (driver.Conn, error) { return &commentTestConn{}, nil }

func (c *commentTestConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (c *commentTestConn) Close() error              { return nil }
func (c *commentTestConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }

func (c *commentTestConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	switch c.step {
	case 0:
		c.step++
		return &commentTestRows{columns: []string{"category"}, data: [][]driver.Value{{nil}}}, nil
	case 1:
		c.step++
		return &commentTestRows{columns: []string{"id", "name", "urls"}, data: [][]driver.Value{{int64(1), "cat", []byte("[\"url\"]")}}}, nil
	case 2:
		c.step++
		data := make([][]driver.Value, len(commentBanned))
		for i, u := range commentBanned {
			data[i] = []driver.Value{u}
		}
		return &commentTestRows{columns: []string{"channel_url"}, data: data}, nil
	default:
		return nil, errors.New("unexpected query")
	}
}

func (c *commentTestConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return commentDummyResult{}, nil
}

func (commentDummyResult) LastInsertId() (int64, error) { return 0, nil }
func (commentDummyResult) RowsAffected() (int64, error) { return 0, nil }

func (r *commentTestRows) Columns() []string { return r.columns }
func (r *commentTestRows) Close() error      { return nil }
func (r *commentTestRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.idx])
	r.idx++
	return nil
}

func init() { sql.Register("commentDummy", commentTestDriver{}) }

func TestPickRandomChannelWithoutCategories(t *testing.T) {
	commentBanned = nil

	db, err := sql.Open("commentDummy", "")
	if err != nil {
		t.Fatalf("не удалось открыть мок БД: %v", err)
	}
	defer func() { _ = db.Close() }()

	cdb := &CommentDB{Conn: db}
	url, err := PickRandomChannel(cdb, 1)
	if err != nil {
		t.Fatalf("ожидался канал, получена ошибка: %v", err)
	}
	if url != "url" {
		t.Fatalf("неверный URL канала: %s", url)
	}
}

func TestPickRandomChannelExcluded(t *testing.T) {
	commentBanned = []string{"url"}

	db, err := sql.Open("commentDummy", "")
	if err != nil {
		t.Fatalf("не удалось открыть мок БД: %v", err)
	}
	defer func() { _ = db.Close() }()

	cdb := &CommentDB{Conn: db}
	_, err = PickRandomChannel(cdb, 1)
	if !errors.Is(err, ErrNoChannel) {
		t.Fatalf("ожидалась ошибка ErrNoChannel, получена: %v", err)
	}
}
