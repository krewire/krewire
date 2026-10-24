package orm_test

import (
	"strings"
	"testing"
	"time"

	"github.com/krewire/krewire/packages/orm"
)

func TestSQLiteConfig_FormatDSN(t *testing.T) {
	// 1. In-memory
	cfgMem := orm.SQLiteConfig{InMemory: true}
	if dsn := cfgMem.FormatDSN(); dsn != ":memory:" {
		t.Errorf("expected :memory:, got %s", dsn)
	}

	// 2. File with WAL and pragmas
	cfgFile := orm.SQLiteConfig{
		Path:        "data/app.db",
		JournalMode: "WAL",
		ForeignKeys: true,
		BusyTimeout: 3 * time.Second,
	}
	dsn := cfgFile.FormatDSN()
	if !strings.HasPrefix(dsn, "data/app.db?") {
		t.Errorf("expected prefix data/app.db?, got %s", dsn)
	}
	if !strings.Contains(dsn, "_journal_mode=WAL") {
		t.Errorf("missing journal mode WAL in DSN: %s", dsn)
	}
	if !strings.Contains(dsn, "_busy_timeout=3000") {
		t.Errorf("missing busy timeout 3000 in DSN: %s", dsn)
	}
}

func TestPostgresConfig_FormatDSN(t *testing.T) {
	cfg := orm.PostgresConfig{
		Host:     "db.internal",
		Port:     5432,
		Database: "krewire_prod",
		User:     "krewire_user",
		Password: "supersecret&password",
		SSLMode:  "require",
		TimeZone: "Asia/Jakarta",
	}

	dsn := cfg.FormatDSN()
	expectedPrefix := "postgres://krewire_user:supersecret%26password@db.internal:5432/krewire_prod"
	if !strings.HasPrefix(dsn, expectedPrefix) {
		t.Errorf("expected prefix %s, got %s", expectedPrefix, dsn)
	}
	if !strings.Contains(dsn, "sslmode=require") {
		t.Errorf("missing sslmode=require in %s", dsn)
	}
	if !strings.Contains(dsn, "timezone=Asia%2FJakarta") {
		t.Errorf("missing timezone in %s", dsn)
	}
}

func TestMySQLConfig_FormatDSN(t *testing.T) {
	cfg := orm.MySQLConfig{
		Host:      "127.0.0.1",
		Port:      3306,
		Database:  "krewire_app",
		User:      "app_user",
		Password:  "db_pass_123",
		Charset:   "utf8mb4",
		Collation: "utf8mb4_unicode_ci",
	}

	dsn := cfg.FormatDSN()
	expectedPrefix := "app_user:db_pass_123@tcp(127.0.0.1:3306)/krewire_app?"
	if !strings.HasPrefix(dsn, expectedPrefix) {
		t.Errorf("expected prefix %s, got %s", expectedPrefix, dsn)
	}
	if !strings.Contains(dsn, "charset=utf8mb4") {
		t.Errorf("missing charset in %s", dsn)
	}
	if !strings.Contains(dsn, "collation=utf8mb4_unicode_ci") {
		t.Errorf("missing collation in %s", dsn)
	}
}
