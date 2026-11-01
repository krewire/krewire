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

func TestQueryBuilder_RejectsUnapprovedOperator(t *testing.T) {
	b := orm.NewQueryBuilder(&orm.SQLiteDialect{})
	b.Table("users").Where("status", "INJECTED_OP", "active")
	sql, args := b.BuildSelect()
	if b.Error() == nil {
		t.Fatal("expected error for unapproved operator, got nil")
	}
	if sql != "" || args != nil {
		t.Fatalf("expected empty query and nil args on error, got sql=%q args=%v", sql, args)
	}

	_, _, err := b.Compile()
	if err == nil {
		t.Fatal("expected Compile() to return error for unapproved operator")
	}

	b2 := orm.NewQueryBuilder(&orm.SQLiteDialect{})
	b2.Table("users").Where("status", "=", "active").OrWhere("role", "DROP TABLE", "admin")
	if err := b2.Error(); err == nil {
		t.Fatal("expected error for unapproved operator in OrWhere, got nil")
	}
}

func TestQueryBuilder_RejectsMalformedIdentifiers(t *testing.T) {
	badColumns := []string{
		"users; DROP TABLE users; --",
		"id OR 1=1",
		"col with space",
		"col'--",
		"col`--",
	}
	for _, col := range badColumns {
		b := orm.NewQueryBuilder(&orm.SQLiteDialect{})
		b.Table("users").Where(col, "=", "val")
		if b.Error() == nil {
			t.Errorf("expected error for malformed column %q, got nil", col)
		}
		sql, _ := b.BuildSelect()
		if sql != "" {
			t.Errorf("expected empty SQL for malformed column %q, got %q", col, sql)
		}
	}

	badTables := []string{
		"users; DROP TABLE users; --",
		"users WHERE 1=1",
	}
	for _, tbl := range badTables {
		b := orm.NewQueryBuilder(&orm.SQLiteDialect{})
		b.Table(tbl).Select("id")
		sql, _ := b.BuildSelect()
		if b.Error() == nil {
			t.Errorf("expected error for malformed table %q, got nil", tbl)
		}
		if sql != "" {
			t.Errorf("expected empty SQL for malformed table %q, got %q", tbl, sql)
		}
	}
}

func TestQueryBuilder_ApprovedOperatorsAndSafeQuoting(t *testing.T) {
	ops := []string{"=", "!=", "<>", "<", "<=", ">", ">=", "LIKE", "ILIKE", "IS", "IS NOT", "BETWEEN"}
	for _, op := range ops {
		b := orm.NewQueryBuilder(&orm.SQLiteDialect{})
		b.Table("users").Where("status", op, "val")
		if err := b.Error(); err != nil {
			t.Errorf("expected operator %q to be allowed, got error: %v", op, err)
		}
		sql, args := b.BuildSelect()
		if sql == "" || len(args) != 1 {
			t.Errorf("expected valid sql for operator %q, got sql=%q", op, sql)
		}
	}
}
