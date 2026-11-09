package migrate_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/krewire/krewire/pkg/orm"
	"github.com/krewire/krewire/pkg/orm/migrate"
	"github.com/krewire/krewire/pkg/orm/sqlite"
)

func file(up, down string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte("-- generated header\n\n-- +up\n" + up + "\n\n-- +down\n" + down + "\n")}
}

func baseFS() fstest.MapFS {
	return fstest.MapFS{
		"20260101000000_create_users.sql": file(
			"CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT NOT NULL);",
			"DROP TABLE users;"),
		"20260102000000_create_posts.sql": file(
			"CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id));\nCREATE INDEX posts_user ON posts(user_id);",
			"DROP TABLE posts;"),
	}
}

func openDB(t *testing.T, path string) *orm.DB {
	t.Helper()
	db, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newMigrator(t *testing.T, db *orm.DB, fsys fstest.MapFS) *migrate.Migrator {
	t.Helper()
	m, err := migrate.New(db, fsys, migrate.Options{LockTimeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func tableExists(t *testing.T, db *orm.DB, name string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM sqlite_master WHERE type IN ('table','index') AND name = ?", name).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n > 0
}

func states(t *testing.T, m *migrate.Migrator) []migrate.State {
	t.Helper()
	st, err := m.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	out := make([]migrate.State, len(st))
	for i, s := range st {
		out[i] = s.State
	}
	return out
}

func TestUpStatusDown(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, filepath.Join(t.TempDir(), "app.db"))
	m := newMigrator(t, db, baseFS())

	if got := states(t, m); len(got) != 2 || got[0] != migrate.StatePending || got[1] != migrate.StatePending {
		t.Fatalf("fresh status = %v", got)
	}
	if tableExists(t, db, migrate.DefaultTable) {
		t.Fatal("Status must not create the bookkeeping table")
	}

	res, err := m.Up(ctx, 1)
	if err != nil || len(res) != 1 || res[0].Migration.Name != "create_users" || res[0].Direction != migrate.DirectionUp {
		t.Fatalf("Up(1) = %+v, %v", res, err)
	}
	if res, err = m.Up(ctx, 0); err != nil || len(res) != 1 || res[0].Migration.Name != "create_posts" {
		t.Fatalf("Up(all) = %+v, %v", res, err)
	}
	if res, err = m.Up(ctx, 0); err != nil || len(res) != 0 {
		t.Fatalf("second Up must be a no-op, got %+v, %v", res, err)
	}
	st, _ := m.Status(ctx)
	if st[0].State != migrate.StateApplied || st[1].State != migrate.StateApplied || st[1].AppliedAt.IsZero() {
		t.Fatalf("status after up = %+v", st)
	}
	if !tableExists(t, db, "posts_user") {
		t.Fatal("multi-statement up section was not fully applied")
	}

	if res, err = m.Down(ctx, 1); err != nil || len(res) != 1 || res[0].Migration.Name != "create_posts" || res[0].Direction != migrate.DirectionDown {
		t.Fatalf("Down(1) = %+v, %v", res, err)
	}
	if tableExists(t, db, "posts") || !tableExists(t, db, "users") {
		t.Fatal("Down(1) must drop only the newest migration")
	}
	if res, err = m.Down(ctx, 5); err != nil || len(res) != 1 {
		t.Fatalf("Down past the first migration must stop cleanly: %+v, %v", res, err)
	}
	if _, err := m.Down(ctx, 0); err == nil {
		t.Fatal("Down(0) must be a usage error")
	}
	if got := states(t, m); got[0] != migrate.StatePending || got[1] != migrate.StatePending {
		t.Fatalf("status after full rollback = %v", got)
	}
}

func TestFailedMigrationRollsBack(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, filepath.Join(t.TempDir(), "app.db"))
	fsys := baseFS()
	fsys["20260103000000_broken.sql"] = file(
		"CREATE TABLE half (id INTEGER);\nINSERT INTO no_such_table VALUES (1);",
		"DROP TABLE half;")
	m := newMigrator(t, db, fsys)

	res, err := m.Up(ctx, 0)
	if err == nil || len(res) != 2 {
		t.Fatalf("Up = %+v, %v; want two applied then an error", res, err)
	}
	if tableExists(t, db, "half") {
		t.Fatal("the failed migration's first statement was not rolled back")
	}
	if got := states(t, m); got[2] != migrate.StatePending {
		t.Fatalf("failed migration must stay pending, status = %v", got)
	}
}

func TestChecksumMismatchAndMissingFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "app.db")
	db := openDB(t, path)
	if _, err := newMigrator(t, db, baseFS()).Up(ctx, 0); err != nil {
		t.Fatal(err)
	}

	edited := baseFS()
	edited["20260101000000_create_users.sql"] = file(
		"CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT NOT NULL, name TEXT);",
		"DROP TABLE users;")
	edited["20260104000000_more.sql"] = file("CREATE TABLE more (id INTEGER);", "DROP TABLE more;")
	m := newMigrator(t, db, edited)
	if _, err := m.Up(ctx, 0); !errors.Is(err, migrate.ErrChecksumMismatch) {
		t.Fatalf("Up over an edited file = %v, want ErrChecksumMismatch", err)
	}
	if tableExists(t, db, "more") {
		t.Fatal("no migration may run while an applied file is edited")
	}
	if _, err := m.Down(ctx, 1); !errors.Is(err, migrate.ErrChecksumMismatch) {
		t.Fatalf("Down over an edited file = %v", err)
	}
	if got := states(t, m); got[0] != migrate.StateModified || got[2] != migrate.StatePending {
		t.Fatalf("status = %v", got)
	}

	missing := baseFS()
	delete(missing, "20260102000000_create_posts.sql")
	m = newMigrator(t, db, missing)
	if got := states(t, m); len(got) != 2 || got[1] != migrate.StateMissing {
		t.Fatalf("status with a deleted file = %v", got)
	}
	if _, err := m.Down(ctx, 1); !errors.Is(err, migrate.ErrMissingFile) {
		t.Fatalf("Down of a deleted file = %v, want ErrMissingFile", err)
	}
}

func TestIrreversible(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, filepath.Join(t.TempDir(), "app.db"))
	fsys := fstest.MapFS{"1_seed.sql": file("CREATE TABLE seed (id INTEGER);", "-- Write your DOWN migration SQL here:")}
	m := newMigrator(t, db, fsys)
	if _, err := m.Up(ctx, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Down(ctx, 1); !errors.Is(err, migrate.ErrIrreversible) {
		t.Fatalf("Down without a down section = %v", err)
	}
	if !tableExists(t, db, "seed") {
		t.Fatal("refused rollback must leave the schema untouched")
	}
}

func TestRedo(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, filepath.Join(t.TempDir(), "app.db"))
	m := newMigrator(t, db, baseFS())
	if _, err := m.Up(ctx, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, "INSERT INTO users (email) VALUES ('a@b.c')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, "INSERT INTO posts (user_id) VALUES (1)"); err != nil {
		t.Fatal(err)
	}
	res, err := m.Redo(ctx)
	if err != nil || len(res) != 2 || res[0].Direction != migrate.DirectionDown || res[1].Direction != migrate.DirectionUp ||
		res[1].Migration.Name != "create_posts" {
		t.Fatalf("Redo = %+v, %v", res, err)
	}
	var n int
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM posts").Scan(&n); err != nil || n != 0 {
		t.Fatalf("redo must recreate posts empty: n=%d err=%v", n, err)
	}
}

// TestRedoIsAtomic makes the second application of a migration fail and
// checks that the rollback half of the redo is undone with it.
func TestRedoIsAtomic(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, filepath.Join(t.TempDir(), "app.db"))
	if _, err := db.Exec(ctx, "CREATE TABLE once (tag TEXT UNIQUE)"); err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{"1_t.sql": file("INSERT INTO once VALUES ('t');\nCREATE TABLE t (id INTEGER);", "DROP TABLE t;")}
	m := newMigrator(t, db, fsys)
	if _, err := m.Up(ctx, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Redo(ctx); err == nil {
		t.Fatal("redo should fail on the UNIQUE constraint")
	}
	if !tableExists(t, db, "t") {
		t.Fatal("failed redo dropped the table: the down half was not rolled back")
	}
	if got := states(t, m); got[0] != migrate.StateApplied {
		t.Fatalf("status after failed redo = %v", got)
	}
}

// TestConcurrentRunners starts several runners on the same file, each with
// its own connection pool as separate processes would have. Every migration
// must be applied exactly once and no runner may fail.
func TestConcurrentRunners(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "app.db")
	fsys := baseFS()
	fsys["20260105000000_count.sql"] = file("CREATE TABLE runs (n INTEGER);\nINSERT INTO runs VALUES (1);", "DROP TABLE runs;")

	// Create the file in WAL mode first: switching a brand-new file to WAL
	// needs an exclusive lock that SQLite does not wait for.
	if err := openDB(t, path).Ping(ctx); err != nil {
		t.Fatal(err)
	}

	const runners = 8
	var wg sync.WaitGroup
	applied := make([]int, runners)
	errc := make(chan error, runners)
	for i := 0; i < runners; i++ {
		db := openDB(t, path)
		m := newMigrator(t, db, fsys)
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := m.Up(ctx, 0)
			applied[i] = len(res)
			errc <- err
		}(i)
	}
	wg.Wait()
	close(errc)
	for err := range errc {
		if err != nil {
			t.Fatalf("concurrent runner failed: %v", err)
		}
	}
	total := 0
	for _, n := range applied {
		total += n
	}
	if total != 3 {
		t.Fatalf("runners applied %d migrations in total, want exactly 3", total)
	}
	db := openDB(t, path)
	var n int
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM runs").Scan(&n); err != nil || n != 1 {
		t.Fatalf("runs has %d rows (err %v): a migration ran more than once", n, err)
	}
}

// TestLockTimeout holds the database write lock from another connection and
// checks the runner gives up with ErrLocked instead of hanging.
func TestLockTimeout(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "app.db")
	holder := openDB(t, path)
	tx, err := holder.Begin(ctx) // immediate: sqlite.Open defaults _txlock=immediate
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()

	m, err := migrate.New(openDB(t, path), baseFS(), migrate.Options{LockTimeout: 200 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := m.Up(ctx, 0); !errors.Is(err, migrate.ErrLocked) {
		t.Fatalf("Up under a held lock = %v, want ErrLocked", err)
	}
	if waited := time.Since(start); waited < 150*time.Millisecond || waited > 5*time.Second {
		t.Fatalf("waited %s for a 200ms lock timeout", waited)
	}
	_ = tx.Rollback()
	if _, err := m.Up(ctx, 0); err != nil {
		t.Fatalf("Up after the lock was released: %v", err)
	}
}

func TestUpHelperAndOptions(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, ":memory:") // one pooled connection: the runner must not deadlock
	if err := migrate.Up(ctx, db, baseFS()); err != nil {
		t.Fatal(err)
	}
	if !tableExists(t, db, "posts") {
		t.Fatal("migrate.Up did not apply the migrations")
	}

	custom, err := migrate.New(db, baseFS(), migrate.Options{Table: "app_migrations"})
	if err != nil {
		t.Fatal(err)
	}
	if got := states(t, custom); got[0] != migrate.StatePending {
		t.Fatalf("a different table tracks its own state, got %v", got)
	}
	if _, err := migrate.New(db, baseFS(), migrate.Options{Table: "x; DROP TABLE users"}); err == nil {
		t.Fatal("unsafe table name accepted")
	}
	if _, err := migrate.New(nil, baseFS(), migrate.Options{}); err == nil {
		t.Fatal("nil database accepted")
	}
}
