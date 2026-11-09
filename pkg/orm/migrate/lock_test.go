package migrate

import (
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/orm"
)

func TestDialectSQL(t *testing.T) {
	pg, err := dialectFor(&orm.PostgresDialect{}, "schema_migrations")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pg.insert, "VALUES ($1, $2, $3, $4)") || !strings.Contains(pg.create, "TIMESTAMPTZ") ||
		!strings.Contains(pg.delete, "version = $1") || !strings.Contains(pg.tableExists, "$1") {
		t.Errorf("postgres SQL: %+v", pg)
	}
	lite, err := dialectFor(&orm.SQLiteDialect{}, "schema_migrations")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lite.insert, "VALUES (?, ?, ?, ?)") || !strings.Contains(lite.create, `"schema_migrations"`) {
		t.Errorf("sqlite SQL: %+v", lite)
	}
	if pg.lockKey != fnvKey("schema_migrations") || pg.lockKey == fnvKey("other") {
		t.Error("advisory lock key must be stable per table and differ between tables")
	}
	if _, err := dialectFor(&orm.MySQLDialect{}, "schema_migrations"); err == nil {
		t.Error("mysql must be refused: its DDL is not transactional")
	}
}
