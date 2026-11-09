package commands

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/orm/sqlite"
)

func runMigrate(t *testing.T, args ...string) (string, errs.ExitCode) {
	t.Helper()
	return captureStdout(t, func() errs.ExitCode {
		fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
		RegisterMigrate(fs)
		if err := fs.Parse(args); err != nil {
			t.Fatal(err)
		}
		return RunMigrate(fs)
	})
}

// fillMigration replaces the generated placeholder sections of the newest
// migration whose name ends in suffix.
func fillMigration(t *testing.T, dir, suffix, up, down string) {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(dir, "*_"+suffix+".sql"))
	if len(matches) != 1 {
		t.Fatalf("expected one *_%s.sql in %s, got %v", suffix, dir, matches)
	}
	src, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "-- +up") || !strings.Contains(string(src), "-- +down") {
		t.Fatalf("generated migration lacks markers:\n%s", src)
	}
	text := strings.Replace(string(src), "-- Write your UP migration SQL here:", up, 1)
	text = strings.Replace(text, "-- Write your DOWN migration SQL here:", down, 1)
	writeFile(t, matches[0], text)
}

// TestMigrate_EndToEndInNewApp scaffolds a project with `kiw new --app`,
// creates migrations with `kiw migrate create`, and drives up, status, down,
// and redo against the SQLite file named in krewire.yaml.
func TestMigrate_EndToEndInNewApp(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	fsNew := flag.NewFlagSet("new", flag.ContinueOnError)
	RegisterNew(fsNew)
	_ = fsNew.Parse([]string{"--app", "--no-input", "shop"})
	if _, code := captureStdout(t, func() errs.ExitCode { return RunNew(fsNew) }); code != errs.ExitCodeSuccess {
		t.Fatalf("kiw new --app = %v", code)
	}
	project := filepath.Join(dir, "shop")
	chdir(t, project)

	if _, code := runMigrate(t, "up"); code != errs.ExitCodeUsage {
		t.Fatalf("up without a configured database = %v, want usage", code)
	}

	yaml, err := os.ReadFile(filepath.Join(project, "krewire.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(project, "krewire.yaml"), string(yaml)+"\ndatabase:\n  driver: sqlite\n  dsn: data/shop.db\n")
	if err := os.MkdirAll(filepath.Join(project, "data"), 0o755); err != nil {
		t.Fatal(err)
	}

	if out, code := runMigrate(t, "create", "create_products"); code != errs.ExitCodeSuccess || !strings.Contains(out, "created migrations/") {
		t.Fatalf("create = %v %q", code, out)
	}
	if _, code := runMigrate(t, "create", "add_price"); code != errs.ExitCodeSuccess {
		t.Fatalf("second create = %v", code)
	}
	migrations := filepath.Join(project, "migrations")
	fillMigration(t, migrations, "create_products", "CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT NOT NULL);", "DROP TABLE products;")
	fillMigration(t, migrations, "add_price", "ALTER TABLE products ADD COLUMN price INTEGER NOT NULL DEFAULT 0;", "ALTER TABLE products DROP COLUMN price;")

	out, code := runMigrate(t, "status")
	if code != errs.ExitCodeSuccess || strings.Count(out, "pending") != 2 {
		t.Fatalf("status before up = %v\n%s", code, out)
	}
	if out, code = runMigrate(t, "up"); code != errs.ExitCodeSuccess || strings.Count(out, "applied ") != 2 {
		t.Fatalf("up = %v\n%s", code, out)
	}
	if out, _ = runMigrate(t, "up"); !strings.Contains(out, "up to date") {
		t.Fatalf("second up = %q", out)
	}

	db, err := sqlite.Open(filepath.Join(project, "data", "shop.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(context.Background(), "INSERT INTO products (name, price) VALUES ('tea', 5)"); err != nil {
		t.Fatalf("schema from kiw migrate up is not usable by the app: %v", err)
	}

	if out, code = runMigrate(t, "redo"); code != errs.ExitCodeSuccess || !strings.Contains(out, "rolled back") || !strings.Contains(out, "add_price") {
		t.Fatalf("redo = %v\n%s", code, out)
	}
	if out, code = runMigrate(t, "down", "2"); code != errs.ExitCodeSuccess || strings.Count(out, "rolled back") != 2 {
		t.Fatalf("down 2 = %v\n%s", code, out)
	}
	if out, _ = runMigrate(t, "down"); !strings.Contains(out, "nothing to roll back") {
		t.Fatalf("down on an empty history = %q", out)
	}

	if _, code = runMigrate(t, "up", "1"); code != errs.ExitCodeSuccess {
		t.Fatalf("up 1 = %v", code)
	}
	edited, _ := filepath.Glob(filepath.Join(migrations, "*_create_products.sql"))
	src, _ := os.ReadFile(edited[0])
	writeFile(t, edited[0], strings.Replace(string(src), "name TEXT NOT NULL", "name TEXT NOT NULL, sku TEXT", 1))
	if out, code = runMigrate(t, "status"); code != errs.ExitCodeFailure || !strings.Contains(out, "modified") {
		t.Fatalf("status over an edited migration = %v\n%s", code, out)
	}
	if _, code = runMigrate(t, "up"); code != errs.ExitCodeFailure {
		t.Fatalf("up over an edited migration = %v, want failure", code)
	}
}

func TestMigrate_ConfigPrecedenceAndErrors(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, filepath.Join(dir, "krewire.yaml"), "project:\n  name: x\n  kind: app\ndatabase:\n  dsn: yaml.db\n  migrations: db/migrations\n")
	writeFile(t, filepath.Join(dir, ".env"), "APP_DATABASE_DSN=env.db\n")
	t.Setenv("APP_DATABASE_DSN", "")
	_ = os.Unsetenv("APP_DATABASE_DSN")

	if out, code := runMigrate(t, "create", "init"); code != errs.ExitCodeSuccess || !strings.Contains(out, "db/migrations/") {
		t.Fatalf("create must honor database.migrations: %v %q", code, out)
	}
	fillMigration(t, filepath.Join(dir, "db", "migrations"), "init", "CREATE TABLE t (id INTEGER);", "DROP TABLE t;")

	if _, code := runMigrate(t, "up"); code != errs.ExitCodeSuccess {
		t.Fatalf("up = %v", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "env.db")); err != nil {
		t.Fatalf(".env APP_DATABASE_DSN must win over krewire.yaml: %v", err)
	}
	if _, code := runMigrate(t, "--dsn", "flag.db", "up"); code != errs.ExitCodeSuccess {
		t.Fatalf("up --dsn = %v", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "flag.db")); err != nil {
		t.Fatalf("--dsn must win over the environment: %v", err)
	}

	for _, args := range [][]string{
		{},
		{"sideways"},
		{"down", "0"},
		{"down", "two"},
		{"status", "3"},
		{"create"},
		{"--driver", "postgres", "--dsn", "postgres://localhost/x", "up"},
		{"--dir", "nope", "up"},
	} {
		if _, code := runMigrate(t, args...); code != errs.ExitCodeUsage {
			t.Errorf("kiw migrate %v = %v, want usage", args, code)
		}
	}
}
