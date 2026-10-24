package orm_test

import (
	"strings"
	"testing"
	"time"

	"github.com/krewire/krewire/packages/orm"
)

type UserAccount struct {
	ID        int64  `orm:"pk;auto"`
	Email     string `orm:"column:email_address;unique;not_null"`
	FullName  string `orm:"column:full_name"`
	IsActive  bool   `orm:"default:1"`
	CreatedAt time.Time
}

func (UserAccount) TableName() string {
	return "custom_users"
}

func TestParseModel_CustomTableAndTags(t *testing.T) {
	tm, err := orm.ParseModel(UserAccount{})
	if err != nil {
		t.Fatalf("ParseModel failed: %v", err)
	}

	if tm.TableName != "custom_users" {
		t.Errorf("expected table name custom_users, got %s", tm.TableName)
	}

	if tm.PrimaryKey == nil || tm.PrimaryKey.Name != "id" || !tm.PrimaryKey.IsAutoInc {
		t.Fatalf("invalid primary key metadata: %+v", tm.PrimaryKey)
	}

	emailCol := tm.ColumnMap["email_address"]
	if emailCol == nil {
		t.Fatal("expected email_address column in map")
	}
	if !emailCol.Unique || emailCol.IsNullable {
		t.Errorf("email column attributes wrong: unique=%v nullable=%v", emailCol.Unique, emailCol.IsNullable)
	}
}

func TestGenerateCreateTableSQL_SQLite(t *testing.T) {
	ddl, err := orm.GenerateCreateTableSQL(UserAccount{}, &orm.SQLiteDialect{})
	if err != nil {
		t.Fatalf("GenerateCreateTableSQL failed: %v", err)
	}

	if !strings.Contains(ddl, `CREATE TABLE IF NOT EXISTS "custom_users"`) {
		t.Errorf("missing create table header: %s", ddl)
	}
	if !strings.Contains(ddl, `"id" INTEGER PRIMARY KEY AUTOINCREMENT`) {
		t.Errorf("missing sqlite autoincrement pk: %s", ddl)
	}
	if !strings.Contains(ddl, `"email_address" TEXT NOT NULL UNIQUE`) {
		t.Errorf("missing unique not null column: %s", ddl)
	}
}

func TestGenerateCreateTableSQL_Postgres(t *testing.T) {
	ddl, err := orm.GenerateCreateTableSQL(UserAccount{}, &orm.PostgresDialect{})
	if err != nil {
		t.Fatalf("GenerateCreateTableSQL failed: %v", err)
	}

	if !strings.Contains(ddl, `"id" BIGSERIAL PRIMARY KEY`) {
		t.Errorf("missing postgres bigserial pk: %s", ddl)
	}
	if !strings.Contains(ddl, `"created_at" TIMESTAMPTZ`) {
		t.Errorf("missing timestamptz column: %s", ddl)
	}
}

func TestGenerateCreateTableSQL_MySQL(t *testing.T) {
	ddl, err := orm.GenerateCreateTableSQL(UserAccount{}, &orm.MySQLDialect{})
	if err != nil {
		t.Fatalf("GenerateCreateTableSQL failed: %v", err)
	}

	if !strings.Contains(ddl, "`id` BIGINT AUTO_INCREMENT PRIMARY KEY") {
		t.Errorf("missing mysql bigint auto_increment pk: %s", ddl)
	}
	if !strings.Contains(ddl, "`custom_users`") {
		t.Errorf("missing mysql backtick quote: %s", ddl)
	}
}
