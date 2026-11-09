package commands

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/orm"
	"github.com/krewire/krewire/pkg/orm/migrate"
	"github.com/krewire/krewire/pkg/orm/sqlite"
	"github.com/krewire/krewire/tools/kiw/internal/config"
	"github.com/krewire/krewire/tools/kiw/internal/scaffold"
)

const migrateUsage = `usage: kiw migrate <command> [flags]

commands:
  up [n]          apply pending migrations (all, or the next n)
  down [n]        roll back the last n applied migrations (default 1)
  redo            roll back the last migration and apply it again
  status          list migrations and their state
  create <name>   create migrations/<timestamp>_<name>.sql (same as kiw make migration)

The database comes from --dsn/--driver, else APP_DATABASE_DSN/APP_DATABASE_DRIVER
(.env is loaded), else database.dsn/database.driver in krewire.yaml.`

// RegisterMigrate registers flags for the migrate command.
func RegisterMigrate(fs *flag.FlagSet) {
	fs.String("dsn", "", "database DSN (overrides APP_DATABASE_DSN and database.dsn)")
	fs.String("driver", "", "database/sql driver (overrides APP_DATABASE_DRIVER and database.driver; default sqlite)")
	fs.String("dir", "", "migrations directory (overrides APP_DATABASE_MIGRATIONS and database.migrations; default migrations)")
	fs.Duration("lock-timeout", migrate.DefaultLockTimeout, "how long to wait for another migration runner")
}

// RunMigrate applies, rolls back, lists, or creates SQL migrations.
func RunMigrate(fs *flag.FlagSet) errs.ExitCode {
	args := fs.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, migrateUsage)
		return errs.ExitCodeUsage
	}
	sub, args := args[0], args[1:]

	root, err := findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kiw: "+err.Error())
		return errs.ExitCodeUsage
	}
	db, err := resolveMigrateDatabase(root, fs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kiw migrate: "+err.Error())
		return errs.ExitCodeUsage
	}

	switch sub {
	case "create", "new":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "kiw migrate: usage: kiw migrate create <name>")
			return errs.ExitCodeUsage
		}
		if _, err := scaffold.MakeMigration(root, db.Migrations, args[0]); err != nil {
			fmt.Fprintln(os.Stderr, "kiw migrate: "+err.Error())
			return errs.ExitCodeFailure
		}
		return errs.ExitCodeSuccess
	case "up", "down", "redo", "status":
	default:
		fmt.Fprintf(os.Stderr, "kiw migrate: unknown command %q\n\n%s\n", sub, migrateUsage)
		return errs.ExitCodeUsage
	}

	steps, err := migrateSteps(sub, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kiw migrate: "+err.Error())
		return errs.ExitCodeUsage
	}
	timeout, _ := time.ParseDuration(flagValue(fs, "lock-timeout"))
	m, closeDB, err := openMigrator(root, db, timeout)
	if err != nil {
		printMigrateError(os.Stderr, err)
		return errs.ExitCodeUsage
	}
	defer closeDB()
	return runMigrateCommand(context.Background(), os.Stdout, m, sub, steps)
}

// migrateSteps parses the optional step count of up and down.
func migrateSteps(sub string, args []string) (int, error) {
	if len(args) == 0 {
		if sub == "down" {
			return 1, nil
		}
		return 0, nil
	}
	if (sub != "up" && sub != "down") || len(args) > 1 {
		return 0, fmt.Errorf("unexpected arguments %v for %q", args, sub)
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 {
		return 0, fmt.Errorf("step count must be a positive number, got %q", args[0])
	}
	return n, nil
}

// resolveMigrateDatabase merges flags, the environment (after .env), and
// krewire.yaml into the effective database settings.
func resolveMigrateDatabase(root string, fs *flag.FlagSet) (config.DatabaseConfig, error) {
	cfg, err := config.Load(root)
	if err != nil {
		return config.DatabaseConfig{}, err
	}
	if err := cfg.LoadDotEnv(root); err != nil {
		return config.DatabaseConfig{}, err
	}
	return cfg.ResolveDatabase(os.LookupEnv, config.DatabaseConfig{
		Driver:     flagValue(fs, "driver"),
		DSN:        flagValue(fs, "dsn"),
		Migrations: flagValue(fs, "dir"),
	})
}

// openMigrator opens the database and loads the migrations directory. kiw
// links only the SQLite driver; other drivers must be registered by the
// process, which for kiw means they are not available.
func openMigrator(root string, db config.DatabaseConfig, timeout time.Duration) (*migrate.Migrator, func(), error) {
	if db.DSN == "" {
		return nil, nil, errs.WithHint(errors.New("no database configured"),
			"set database.dsn in krewire.yaml, APP_DATABASE_DSN in the environment or .env, or pass --dsn")
	}
	dir := db.Migrations
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(root, dir)
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return nil, nil, errs.WithHint(fmt.Errorf("migrations directory %s not found", dir),
			"create a migration first: kiw migrate create <name>")
	}

	var conn *orm.DB
	var err error
	switch {
	case db.IsSQLite():
		conn, err = sqlite.Open(db.SQLitePath(root))
	case slices.Contains(sql.Drivers(), db.Driver):
		conn, err = orm.Open(db.Driver, db.DSN)
	default:
		return nil, nil, errs.WithHint(fmt.Errorf("database driver %q is not built into kiw", db.Driver),
			"kiw bundles only the SQLite driver; for "+db.Driver+", import your driver and call migrate.Up from pkg/orm/migrate in your app")
	}
	if err != nil {
		return nil, nil, err
	}
	m, err := migrate.New(conn, os.DirFS(dir), migrate.Options{LockTimeout: timeout})
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	return m, func() { _ = conn.Close() }, nil
}

// runMigrateCommand runs one migrate sub-command and prints its outcome.
func runMigrateCommand(ctx context.Context, w io.Writer, m *migrate.Migrator, sub string, steps int) errs.ExitCode {
	var results []migrate.Result
	var err error
	switch sub {
	case "status":
		return printMigrateStatus(ctx, w, m)
	case "up":
		results, err = m.Up(ctx, steps)
	case "down":
		results, err = m.Down(ctx, steps)
	case "redo":
		results, err = m.Redo(ctx)
	}
	for _, r := range results {
		verb := "applied"
		if r.Direction == migrate.DirectionDown {
			verb = "rolled back"
		}
		fmt.Fprintf(w, "%s %s (%s)\n", verb, r.Migration, roundDuration(r.Duration))
	}
	if err != nil {
		printMigrateError(os.Stderr, err)
		return errs.ExitCodeFailure
	}
	if len(results) == 0 {
		if sub == "up" {
			fmt.Fprintln(w, "nothing to migrate: the database is up to date")
		} else {
			fmt.Fprintln(w, "nothing to roll back: no migration is applied")
		}
	}
	return errs.ExitCodeSuccess
}

// printMigrateStatus prints one line per migration. It fails when an applied
// migration was edited or deleted, so CI can gate on it.
func printMigrateStatus(ctx context.Context, w io.Writer, m *migrate.Migrator) errs.ExitCode {
	st, err := m.Status(ctx)
	if err != nil {
		printMigrateError(os.Stderr, err)
		return errs.ExitCodeFailure
	}
	if len(st) == 0 {
		fmt.Fprintln(w, "no migrations yet: kiw migrate create <name>")
		return errs.ExitCodeSuccess
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "STATE\tMIGRATION\tAPPLIED AT")
	code := errs.ExitCodeSuccess
	for _, s := range st {
		at := "-"
		if !s.AppliedAt.IsZero() {
			at = s.AppliedAt.UTC().Format(time.RFC3339)
		}
		fmt.Fprintf(tw, "%s\t%s_%s\t%s\n", s.State, s.Version, s.Name, at)
		if s.State == migrate.StateModified || s.State == migrate.StateMissing {
			code = errs.ExitCodeFailure
		}
	}
	_ = tw.Flush()
	return code
}

func printMigrateError(w io.Writer, err error) {
	fmt.Fprintln(w, "kiw migrate: "+err.Error())
	if hint := errs.HintOf(err); hint != "" {
		fmt.Fprintln(w, "hint: "+hint)
	}
}

// roundDuration keeps sub-millisecond timings readable instead of "0s".
func roundDuration(d time.Duration) time.Duration {
	if d < time.Millisecond {
		return d.Round(time.Microsecond)
	}
	return d.Round(time.Millisecond)
}
