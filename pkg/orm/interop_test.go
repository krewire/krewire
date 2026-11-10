package orm_test

import (
	"context"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/orm"
	"github.com/krewire/krewire/pkg/orm/elasticsearch"
	"github.com/krewire/krewire/pkg/orm/mongodb"
	"github.com/krewire/krewire/pkg/orm/mysql"
	"github.com/krewire/krewire/pkg/orm/postgres"
	"github.com/krewire/krewire/pkg/orm/sqlite"
)

type InteropAuditItem struct {
	ID        int64     `orm:"pk;auto"`
	Actor     string    `orm:"column:actor"`
	Action    string    `orm:"column:action"`
	CreatedAt time.Time `orm:"column:created_at"`
}

func (InteropAuditItem) TableName() string {
	return "interop_audit_log"
}

// TestAllFiveDriversArchitecture verifies that all 5 database drivers
// (SQLite, MySQL, PostgreSQL, MongoDB, ElasticSearch) adhere to the
// unified Adapter / DB interfaces and production standards.
func TestAllFiveDriversArchitecture(t *testing.T) {
	ctx := context.Background()

	// 1. SQLite Driver
	sqliteDB, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("sqlite.Open failed: %v", err)
	}
	defer sqliteDB.Close()

	if sqliteDB.Name() != sqlite.DriverName || sqliteDB.Type() != orm.AdapterTypeSQL {
		t.Errorf("unexpected sqlite adapter identity: name=%s type=%s", sqliteDB.Name(), sqliteDB.Type())
	}
	if err := sqliteDB.Ping(ctx); err != nil {
		t.Fatalf("sqlite.Ping failed: %v", err)
	}

	// Migrate and verify CRUD in SQLite
	if err := sqliteDB.AutoMigrate(ctx, &InteropAuditItem{}); err != nil {
		t.Fatalf("sqlite AutoMigrate failed: %v", err)
	}
	insertedID, err := sqliteDB.Model(&InteropAuditItem{}).Insert(&InteropAuditItem{
		Actor:     "agent-zero",
		Action:    "deploy-cluster",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("sqlite Insert failed: %v", err)
	}
	if insertedID <= 0 {
		t.Errorf("expected positive inserted ID, got %d", insertedID)
	}

	// 2. MySQL Driver Config & Dialect
	myCfg := mysql.ProductionConfig("mysql.cluster.local", 3306, "krewire", "pass123", "sdlc_db")
	myDSN := mysql.DSN(myCfg)
	if myDSN == "" || !myCfg.InterpolateParams {
		t.Error("invalid MySQL production configuration")
	}
	myDialect := orm.GetDialect(mysql.DriverName)
	if myDialect.Name() != "mysql" || myDialect.QuoteIdentifier("tbl") != "`tbl`" {
		t.Errorf("unexpected MySQL dialect behavior: %s", myDialect.Name())
	}

	// 3. PostgreSQL Driver Config & Dialect
	pgCfg := postgres.ProductionConfig("pg.cluster.local", 5432, "postgres", "pass123", "sdlc_db")
	pgDSN := postgres.DSN(pgCfg)
	if pgDSN == "" || pgCfg.SSLMode != "require" {
		t.Error("invalid PostgreSQL production configuration")
	}
	pgDialect := orm.GetDialect(postgres.DriverName)
	if pgDialect.Name() != "postgres" || pgDialect.Placeholder(3) != "$3" {
		t.Errorf("unexpected PostgreSQL dialect behavior: %s", pgDialect.Name())
	}

	// 4. MongoDB Driver & Adapter
	mongoAdapter, err := mongodb.Open("memory://", "audit_mongo")
	if err != nil {
		t.Fatalf("mongodb.Open failed: %v", err)
	}
	defer mongoAdapter.Close()

	if mongoAdapter.Name() != "mongodb" || mongoAdapter.Type() != orm.AdapterTypeDocument {
		t.Errorf("unexpected mongodb adapter identity: name=%s type=%s", mongoAdapter.Name(), mongoAdapter.Type())
	}
	if err := mongoAdapter.Ping(ctx); err != nil {
		t.Fatalf("mongodb.Ping failed: %v", err)
	}

	mongoCol := mongoAdapter.Collection("audit_events")
	docID, err := mongoCol.InsertOne(ctx, map[string]interface{}{
		"actor":  "admin",
		"action": "create_user",
		"status": "success",
	})
	if err != nil || docID == "" {
		t.Fatalf("mongo InsertOne failed: %v", err)
	}
	count, err := mongoCol.CountDocuments(ctx, map[string]interface{}{"status": "success"})
	if err != nil || count != 1 {
		t.Errorf("mongo CountDocuments expected 1, got %d (err: %v)", count, err)
	}

	// 5. ElasticSearch Driver & Adapter
	esCfg := elasticsearch.ProductionConfig("http://localhost:9200")
	if len(esCfg.Addresses) != 1 || esCfg.MaxRetries != 3 {
		t.Errorf("unexpected elasticsearch config: %+v", esCfg)
	}
	esAdapter, err := elasticsearch.OpenConfig(esCfg)
	if err != nil {
		t.Fatalf("elasticsearch.OpenConfig failed: %v", err)
	}
	defer esAdapter.Close()

	if esAdapter.Name() != "elasticsearch" || esAdapter.Type() != orm.AdapterTypeSearch {
		t.Errorf("unexpected elasticsearch adapter identity: name=%s type=%s", esAdapter.Name(), esAdapter.Type())
	}
}
