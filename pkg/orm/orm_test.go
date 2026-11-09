package orm_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/krewire/krewire/pkg/orm"
)

var (
	registerOnce sync.Once
)

type mockDriver struct {
	mu          sync.Mutex
	lastQuery   string
	lastArgs    []driver.Value
	rowsToYield [][]driver.Value
	colsToYield []string
}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{drv: d}, nil
}

type mockConn struct {
	drv *mockDriver
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return &mockStmt{drv: c.drv, query: query}, nil
}

func (c *mockConn) Close() error {
	return nil
}

func (c *mockConn) Begin() (driver.Tx, error) {
	return &mockTx{}, nil
}

type mockTx struct{}

func (t *mockTx) Commit() error   { return nil }
func (t *mockTx) Rollback() error { return nil }

type mockStmt struct {
	drv   *mockDriver
	query string
}

func (s *mockStmt) Close() error {
	return nil
}

func (s *mockStmt) NumInput() int {
	return -1
}

func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	s.drv.mu.Lock()
	defer s.drv.mu.Unlock()
	s.drv.lastQuery = s.query
	s.drv.lastArgs = args
	return driver.RowsAffected(1), nil
}

func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	s.drv.mu.Lock()
	defer s.drv.mu.Unlock()
	s.drv.lastQuery = s.query
	s.drv.lastArgs = args

	return &mockRows{
		cols: s.drv.colsToYield,
		rows: s.drv.rowsToYield,
		pos:  0,
	}, nil
}

type mockRows struct {
	cols []string
	rows [][]driver.Value
	pos  int
}

func (r *mockRows) Columns() []string {
	return r.cols
}

func (r *mockRows) Close() error {
	return nil
}

func (r *mockRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.rows) {
		return io.EOF
	}
	row := r.rows[r.pos]
	r.pos++
	for i, val := range row {
		dest[i] = val
	}
	return nil
}

var globalMockDriver = &mockDriver{}

func initMockDriver() {
	registerOnce.Do(func() {
		sql.Register("krewire_mock", globalMockDriver)
	})
}

type Member struct {
	ID    int64  `orm:"pk;auto"`
	Name  string `orm:"column:name"`
	Email string `orm:"column:email"`
}

func TestORM_BasicOperationsWithMock(t *testing.T) {
	initMockDriver()

	db, err := orm.Open("krewire_mock", "mock_dsn")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// 1. Test Insert
	member := &Member{Name: "Alice", Email: "alice@krewire.com"}
	affected, err := db.Table("members").WithContext(ctx).Insert(member)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if affected != 1 {
		t.Errorf("expected 1 row affected, got %d", affected)
	}

	globalMockDriver.mu.Lock()
	if globalMockDriver.lastQuery != `INSERT INTO "members" ("name", "email") VALUES (?, ?)` {
		t.Errorf("unexpected insert query: %s", globalMockDriver.lastQuery)
	}
	globalMockDriver.mu.Unlock()

	// 2. Test Transaction Rollback on error
	txErr := db.Transaction(ctx, func(tx *orm.Tx) error {
		return errors.New("simulated error to trigger rollback")
	})
	if txErr == nil {
		t.Fatal("expected error from transaction, got nil")
	}

	// 3. Test Transaction Commit on success
	err = db.Transaction(ctx, func(tx *orm.Tx) error {
		return nil
	})
	if err != nil {
		t.Fatalf("transaction commit failed: %v", err)
	}
}

func TestORM_FindWithMockRows(t *testing.T) {
	initMockDriver()

	globalMockDriver.mu.Lock()
	globalMockDriver.colsToYield = []string{"id", "name", "email"}
	globalMockDriver.rowsToYield = [][]driver.Value{
		{int64(10), "Bob", "bob@krewire.com"},
	}
	globalMockDriver.mu.Unlock()

	db, err := orm.Open("krewire_mock", "mock_dsn")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	var m Member
	err = db.Table("members").Find(&m, 10)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}

	if m.ID != 10 || m.Name != "Bob" || m.Email != "bob@krewire.com" {
		t.Errorf("scanned struct mismatch: %+v", m)
	}
}

func TestORM_FindAllWithMockRows(t *testing.T) {
	initMockDriver()

	globalMockDriver.mu.Lock()
	globalMockDriver.colsToYield = []string{"id", "name", "email"}
	globalMockDriver.rowsToYield = [][]driver.Value{
		{int64(1), "User 1", "u1@krewire.com"},
		{int64(2), "User 2", "u2@krewire.com"},
	}
	globalMockDriver.mu.Unlock()

	db, err := orm.Open("krewire_mock", "mock_dsn")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	var members []Member
	err = db.Table("members").FindAll(&members)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].Name != "User 1" || members[1].Name != "User 2" {
		t.Errorf("unexpected members slice: %+v", members)
	}
}

func TestTx_ExecQueryInsideTransaction(t *testing.T) {
	initMockDriver()

	db, err := orm.Open("krewire_mock", "mock_dsn")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()
	ctx := context.Background()

	err = db.Transaction(ctx, func(tx *orm.Tx) error {
		if tx.Dialect() == nil || tx.SQLTx() == nil {
			t.Fatal("transaction must expose its dialect and *sql.Tx")
		}
		res, err := tx.Exec(ctx, "UPDATE members SET name = ? WHERE id = ?", "Ana", 7)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			t.Errorf("RowsAffected = %d, want 1", n)
		}
		globalMockDriver.mu.Lock()
		got, args := globalMockDriver.lastQuery, len(globalMockDriver.lastArgs)
		globalMockDriver.colsToYield = []string{"n"}
		globalMockDriver.rowsToYield = [][]driver.Value{{int64(3)}}
		globalMockDriver.mu.Unlock()
		if got != "UPDATE members SET name = ? WHERE id = ?" || args != 2 {
			t.Errorf("Exec ran %q with %d args", got, args)
		}

		var n int64
		if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM members").Scan(&n); err != nil {
			return err
		}
		if n != 3 {
			t.Errorf("QueryRow scanned %d, want 3", n)
		}
		rows, err := tx.Query(ctx, "SELECT COUNT(*) FROM members")
		if err != nil {
			return err
		}
		return rows.Close()
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}
}

func TestTx_NilTransactionIsAFailure(t *testing.T) {
	var tx orm.Tx
	if _, err := tx.Exec(context.Background(), "SELECT 1"); err == nil {
		t.Error("Exec on a nil transaction should fail")
	}
	if _, err := tx.Query(context.Background(), "SELECT 1"); err == nil {
		t.Error("Query on a nil transaction should fail")
	}
	if tx.QueryRow(context.Background(), "SELECT 1") != nil {
		t.Error("QueryRow on a nil transaction should return nil")
	}
}
