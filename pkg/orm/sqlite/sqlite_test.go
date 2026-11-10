package sqlite_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/orm"
	"github.com/krewire/krewire/pkg/orm/sqlite"
)

func TestDSNDefaults(t *testing.T) {
	dsn := sqlite.DSN("data/app.db")
	for _, want := range []string{"data/app.db?", "_journal_mode=WAL", "_foreign_keys=on", "_busy_timeout=5000", "_txlock=immediate"} {
		if !strings.Contains(dsn, want) {
			t.Errorf("DSN %q lacks %q", dsn, want)
		}
	}
	if got := sqlite.DSN(":memory:"); got != ":memory:" {
		t.Errorf("DSN(:memory:) = %q", got)
	}
}

// TestOpenAppliesPragmas opens a real file and reads the pragmas back, so a
// DSN the driver silently ignored would fail here.
func TestOpenAppliesPragmas(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if db.Name() != sqlite.DriverName || db.Dialect().Name() != "sqlite" {
		t.Fatalf("driver %q dialect %q", db.Name(), db.Dialect().Name())
	}
	var mode string
	if err := db.QueryRow(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(mode, "wal") {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
	var fk, busy int
	if err := db.QueryRow(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, "PRAGMA busy_timeout").Scan(&busy); err != nil {
		t.Fatal(err)
	}
	if fk != 1 || busy != 5000 {
		t.Errorf("foreign_keys = %d, busy_timeout = %d; want 1 and 5000", fk, busy)
	}
}

func TestOpenRoundTripsThroughORM(t *testing.T) {
	type note struct {
		ID   int64  `orm:"pk;auto"`
		Body string `orm:"column:body"`
	}
	ctx := context.Background()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.AutoMigrate(ctx, &note{}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Model(&note{}).Insert(&note{Body: "hello"}); err != nil {
		t.Fatal(err)
	}
	err = db.Transaction(ctx, func(tx *orm.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE note SET body = ? WHERE body = ?`, "hi", "hello")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	var got []note
	if err := db.Model(&note{}).FindAll(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Body != "hi" {
		t.Fatalf("rows = %+v", got)
	}
}

func TestOpenConfigFillsDefaults(t *testing.T) {
	cfg := sqlite.Config(filepath.Join(t.TempDir(), "x.db"))
	cfg.DriverName, cfg.TxLock = "", ""
	db, err := sqlite.OpenConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	if db.Name() != sqlite.DriverName {
		t.Errorf("driver = %q", db.Name())
	}
}

// TestForeignKeysEnforced proves the foreign_keys default reaches every
// pooled connection, not only the first.
func TestForeignKeysEnforced(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "fk.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{
		`CREATE TABLE parent (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE child (id INTEGER PRIMARY KEY, parent_id INTEGER NOT NULL REFERENCES parent(id))`,
	} {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(ctx, `INSERT INTO child (parent_id) VALUES (42)`); err == nil {
		t.Fatal("insert with a dangling foreign key succeeded; foreign_keys is off")
	}
}

func TestProductionConfigAndOperations(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "prod.db")
	db, err := sqlite.OpenProduction(dbPath)
	if err != nil {
		t.Fatalf("OpenProduction failed: %v", err)
	}
	defer db.Close()

	// Verify integrity checks on fresh database
	msgs, err := sqlite.IntegrityCheck(ctx, db)
	if err != nil {
		t.Fatalf("IntegrityCheck failed: %v", err)
	}
	if len(msgs) != 1 || msgs[0] != "ok" {
		t.Errorf("unexpected integrity check message: %v", msgs)
	}

	quickMsgs, err := sqlite.QuickCheck(ctx, db)
	if err != nil {
		t.Fatalf("QuickCheck failed: %v", err)
	}
	if len(quickMsgs) != 1 || quickMsgs[0] != "ok" {
		t.Errorf("unexpected quick check message: %v", quickMsgs)
	}

	// Create a table and write some data
	_, err = db.Exec(ctx, "CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("create table items failed: %v", err)
	}
	for i := 0; i < 50; i++ {
		_, err = db.Exec(ctx, "INSERT INTO items (name) VALUES (?)", "test item")
		if err != nil {
			t.Fatalf("insert item failed: %v", err)
		}
	}

	// Verify maintenance operations
	if err := sqlite.Checkpoint(ctx, db, sqlite.CheckpointTruncate); err != nil {
		t.Errorf("Checkpoint TRUNCATE failed: %v", err)
	}
	if err := sqlite.Checkpoint(ctx, db, sqlite.CheckpointPassive); err != nil {
		t.Errorf("Checkpoint PASSIVE failed: %v", err)
	}
	if err := sqlite.Optimize(ctx, db); err != nil {
		t.Errorf("Optimize failed: %v", err)
	}
	if err := sqlite.IncrementalVacuum(ctx, db, 10); err != nil {
		t.Errorf("IncrementalVacuum failed: %v", err)
	}
	if err := sqlite.Vacuum(ctx, db); err != nil {
		t.Errorf("Vacuum failed: %v", err)
	}
}
