package schema_test

import (
	"context"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/orm"
	"github.com/krewire/krewire/pkg/orm/schema"
	"github.com/krewire/krewire/pkg/orm/sqlite"
)

func TestSchemaBuilder_CompileCreate(t *testing.T) {
	bSQLite := schema.NewTxBuilder(context.Background(), nil, &orm.SQLiteDialect{})
	bMySQL := schema.NewTxBuilder(context.Background(), nil, &orm.MySQLDialect{})
	bPG := schema.NewTxBuilder(context.Background(), nil, &orm.PostgresDialect{})

	bp := schema.NewBlueprint("users", true)
	bp.ID()
	bp.String("name", 150).NotNull()
	bp.String("email", 255).Unique()
	bp.Boolean("is_active").Default(true)
	bp.RememberToken()
	bp.Timestamps()

	// 1. SQLite Compile
	sqlLite, _, err := bSQLite.CompileCreate(bp)
	if err != nil {
		t.Fatalf("CompileCreate SQLite failed: %v", err)
	}
	if !strings.Contains(sqlLite, `"id" INTEGER PRIMARY KEY AUTOINCREMENT`) {
		t.Errorf("missing SQLite autoincrement in:\n%s", sqlLite)
	}
	if !strings.Contains(sqlLite, `"name" TEXT NOT NULL`) {
		t.Errorf("missing name TEXT in:\n%s", sqlLite)
	}
	if !strings.Contains(sqlLite, `"email" TEXT`) || !strings.Contains(sqlLite, `UNIQUE`) {
		t.Errorf("missing email UNIQUE in:\n%s", sqlLite)
	}

	// 2. MySQL Compile
	sqlMySQL, _, err := bMySQL.CompileCreate(bp)
	if err != nil {
		t.Fatalf("CompileCreate MySQL failed: %v", err)
	}
	if !strings.Contains(sqlMySQL, "`id` BIGINT AUTO_INCREMENT PRIMARY KEY") {
		t.Errorf("missing MySQL autoincrement in:\n%s", sqlMySQL)
	}
	if !strings.Contains(sqlMySQL, "`name` VARCHAR(150) NOT NULL") {
		t.Errorf("missing name VARCHAR(150) in:\n%s", sqlMySQL)
	}
	if !strings.Contains(sqlMySQL, "`is_active` TINYINT(1)") {
		t.Errorf("missing is_active TINYINT(1) in:\n%s", sqlMySQL)
	}

	// 3. PostgreSQL Compile
	sqlPG, _, err := bPG.CompileCreate(bp)
	if err != nil {
		t.Fatalf("CompileCreate PG failed: %v", err)
	}
	if !strings.Contains(sqlPG, `"id" BIGSERIAL PRIMARY KEY`) {
		t.Errorf("missing PG bigserial in:\n%s", sqlPG)
	}
	if !strings.Contains(sqlPG, `"is_active" BOOLEAN`) {
		t.Errorf("missing is_active BOOLEAN in:\n%s", sqlPG)
	}
	if !strings.Contains(sqlPG, `"created_at" TIMESTAMPTZ`) {
		t.Errorf("missing created_at TIMESTAMPTZ in:\n%s", sqlPG)
	}
}

func TestSchemaBuilder_RealExecutionSQLite(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("sqlite.Open failed: %v", err)
	}
	defer db.Close()

	s := schema.NewBuilder(ctx, db)

	// 1. Create table with Laravel Blueprint syntax
	err = s.Create("flights", func(table *schema.Blueprint) {
		table.ID()
		table.String("flight_number", 50).NotNull().Unique()
		table.String("destination", 100).NotNull()
		table.Integer("capacity").Default(200)
		table.Boolean("is_delayed").Default(false)
		table.Timestamps()
		table.Index("dest_idx", "destination")
	})
	if err != nil {
		t.Fatalf("s.Create failed: %v", err)
	}

	// 2. Check HasTable and HasColumn
	hasFlights, err := s.HasTable("flights")
	if err != nil || !hasFlights {
		t.Fatalf("expected HasTable flights true, got %v (err: %v)", hasFlights, err)
	}

	hasFlightNumber, err := s.HasColumn("flights", "flight_number")
	if err != nil || !hasFlightNumber {
		t.Fatalf("expected HasColumn flight_number true, got %v (err: %v)", hasFlightNumber, err)
	}

	hasBogus, err := s.HasColumn("flights", "non_existing_col")
	if err != nil || hasBogus {
		t.Fatalf("expected HasColumn non_existing_col false, got %v (err: %v)", hasBogus, err)
	}

	// 3. Alter Table (add column)
	err = s.Table("flights", func(table *schema.Blueprint) {
		table.String("airline", 100).Nullable()
	})
	if err != nil {
		t.Fatalf("s.Table alter failed: %v", err)
	}

	hasAirline, err := s.HasColumn("flights", "airline")
	if err != nil || !hasAirline {
		t.Fatalf("expected HasColumn airline true after alter, got %v", hasAirline)
	}

	// 4. Drop table if exists
	if err := s.DropIfExists("flights"); err != nil {
		t.Fatalf("DropIfExists failed: %v", err)
	}

	hasAfterDrop, _ := s.HasTable("flights")
	if hasAfterDrop {
		t.Error("expected table flights to not exist after drop")
	}
}
