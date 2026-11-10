package postgres_test

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/orm"
	"github.com/krewire/krewire/pkg/orm/postgres"
)

func TestDriverRegistration(t *testing.T) {
	var found bool
	for _, d := range sql.Drivers() {
		if d == postgres.DriverName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("driver %q was not registered in database/sql", postgres.DriverName)
	}
}

func TestConfigAndDSN(t *testing.T) {
	cfg := postgres.Config("pg.production.local", 5433, "krew_app", "p@ssword!#", "krewire_iam")
	cfg.SearchPath = "krewire_auth,public"
	cfg.ApplicationName = "krewire-auth-server"
	cfg.ConnectTimeout = 8 * time.Second
	cfg.StatementTimeout = 25 * time.Second

	dsn := postgres.DSN(cfg)

	expectedPrefix := "postgres://krew_app:p%40ssword%21%23@pg.production.local:5433/krewire_iam?"
	if !strings.HasPrefix(dsn, expectedPrefix) {
		t.Errorf("expected DSN prefix %s, got %s", expectedPrefix, dsn)
	}

	for _, token := range []string{
		"sslmode=disable",
		"timezone=UTC",
		"search_path=krewire_auth%2Cpublic",
		"application_name=krewire-auth-server",
		"connect_timeout=8",
		"statement_timeout=25000",
	} {
		if !strings.Contains(dsn, token) {
			t.Errorf("DSN missing expected token %q: %s", token, dsn)
		}
	}
}

func TestProductionConfig(t *testing.T) {
	prod := postgres.ProductionConfig("db-primary.internal", 0, "admin", "adm_pass", "krewire_cloud")
	if prod.Port != 5432 {
		t.Errorf("expected default port 5432, got %d", prod.Port)
	}
	if prod.SSLMode != "require" {
		t.Errorf("expected SSLMode require in production, got %s", prod.SSLMode)
	}
	if prod.MaxOpenConns != 50 {
		t.Errorf("expected MaxOpenConns 50, got %d", prod.MaxOpenConns)
	}
	if prod.MaxIdleConns != 15 {
		t.Errorf("expected MaxIdleConns 15, got %d", prod.MaxIdleConns)
	}
	if prod.ConnMaxLifetime != 15*time.Minute {
		t.Errorf("expected ConnMaxLifetime 15m, got %v", prod.ConnMaxLifetime)
	}
}

func TestPostgresDialectCompatibility(t *testing.T) {
	dialect := orm.GetDialect(postgres.DriverName)
	if dialect.Name() != "postgres" {
		t.Errorf("expected dialect postgres, got %s", dialect.Name())
	}
	if quoted := dialect.QuoteIdentifier("accounts"); quoted != `"accounts"` {
		t.Errorf("expected \"accounts\", got %s", quoted)
	}
	if ph1 := dialect.Placeholder(1); ph1 != "$1" {
		t.Errorf("expected $1, got %s", ph1)
	}
	if ph5 := dialect.Placeholder(5); ph5 != "$5" {
		t.Errorf("expected $5, got %s", ph5)
	}
	if pk := dialect.AutoIncrementPrimaryKeyClause(); pk != "BIGSERIAL PRIMARY KEY" {
		t.Errorf("unexpected PK clause: %s", pk)
	}
}
