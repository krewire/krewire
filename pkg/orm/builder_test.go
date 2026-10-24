package orm_test

import (
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/orm"
)

func TestQueryBuilder_SelectSQLite(t *testing.T) {
	b := orm.NewQueryBuilder(&orm.SQLiteDialect{})
	b.Table("users").
		Select("id", "name", "email").
		Where("status", "=", "active").
		WhereIn("role", "admin", "owner").
		OrderBy("created_at", "DESC").
		Limit(10).
		Offset(20)

	sql, args := b.BuildSelect()

	expectedSQL := `SELECT "id", "name", "email" FROM "users" WHERE "status" = ? AND "role" IN (?, ?) ORDER BY "created_at" DESC LIMIT 10 OFFSET 20`
	if sql != expectedSQL {
		t.Errorf("expected SQL:\n%s\ngot:\n%s", expectedSQL, sql)
	}

	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(args))
	}
	if args[0] != "active" || args[1] != "admin" || args[2] != "owner" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestQueryBuilder_SelectPostgres(t *testing.T) {
	b := orm.NewQueryBuilder(&orm.PostgresDialect{})
	b.Table("users").
		Select("id", "email").
		Where("status", "=", "active").
		Where("is_verified", "=", true).
		OrderBy("id", "ASC").
		Limit(5)

	sql, args := b.BuildSelect()

	expectedSQL := `SELECT "id", "email" FROM "users" WHERE "status" = $1 AND "is_verified" = $2 ORDER BY "id" ASC LIMIT 5`
	if sql != expectedSQL {
		t.Errorf("expected SQL:\n%s\ngot:\n%s", expectedSQL, sql)
	}

	if len(args) != 2 || args[0] != "active" || args[1] != true {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestQueryBuilder_SelectMySQL(t *testing.T) {
	b := orm.NewQueryBuilder(&orm.MySQLDialect{})
	b.Table("orders").
		Select("id", "total_price").
		Where("status", "=", "completed").
		OrWhere("status", "=", "shipped").
		Limit(25)

	sql, args := b.BuildSelect()

	expectedSQL := "SELECT `id`, `total_price` FROM `orders` WHERE `status` = ? OR `status` = ? LIMIT 25"
	if sql != expectedSQL {
		t.Errorf("expected SQL:\n%s\ngot:\n%s", expectedSQL, sql)
	}

	if len(args) != 2 || args[0] != "completed" || args[1] != "shipped" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestQueryBuilder_InsertAndDelete(t *testing.T) {
	b := orm.NewQueryBuilder(&orm.SQLiteDialect{})
	b.Table("products")

	insertSQL := b.BuildInsert([]string{"name", "price", "stock"})
	expectedInsert := `INSERT INTO "products" ("name", "price", "stock") VALUES (?, ?, ?)`
	if insertSQL != expectedInsert {
		t.Errorf("insert SQL mismatch: got %s, want %s", insertSQL, expectedInsert)
	}

	delSQL, delArgs := b.Where("id", "=", 42).BuildDelete()
	expectedDel := `DELETE FROM "products" WHERE "id" = ?`
	if delSQL != expectedDel {
		t.Errorf("delete SQL mismatch: got %s, want %s", delSQL, expectedDel)
	}
	if len(delArgs) != 1 || delArgs[0] != 42 {
		t.Errorf("unexpected del args: %v", delArgs)
	}
}

func TestQueryBuilder_JoinsAndDistinct(t *testing.T) {
	b := orm.NewQueryBuilder(&orm.PostgresDialect{})
	b.Table("orders").
		Distinct().
		Select("orders.id", "customers.name").
		InnerJoin("customers", "orders.customer_id = customers.id").
		Where("orders.total", ">", 1000)

	sql, _ := b.BuildSelect()
	if !strings.Contains(sql, "SELECT DISTINCT") {
		t.Error("expected SELECT DISTINCT")
	}
	if !strings.Contains(sql, `INNER JOIN "customers" ON orders.customer_id = customers.id`) {
		t.Errorf("missing or malformed join: %s", sql)
	}
}
