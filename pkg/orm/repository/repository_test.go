package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/krewire/krewire/pkg/orm"
	"github.com/krewire/krewire/pkg/orm/repository"
	"github.com/krewire/krewire/pkg/orm/sqlite"
)

type Product struct {
	orm.BaseModel
	Name     string  `orm:"column:name"`
	SKU      string  `orm:"column:sku;unique"`
	Price    float64 `orm:"column:price"`
	InStock  bool    `orm:"column:in_stock"`
	Category string  `orm:"column:category"`
}

func (Product) TableName() string {
	return "products"
}

func setupTestDB(t *testing.T) (*orm.DB, func()) {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	createTableSQL := `
	CREATE TABLE products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		sku TEXT NOT NULL UNIQUE,
		price REAL NOT NULL,
		in_stock INTEGER NOT NULL DEFAULT 1,
		category TEXT NOT NULL,
		created_at TIMESTAMP,
		updated_at TIMESTAMP,
		deleted_at TIMESTAMP
	);`

	if _, err := db.Exec(context.Background(), createTableSQL); err != nil {
		db.Close()
		t.Fatalf("failed to create table: %v", err)
	}

	return db, func() {
		_ = db.Close()
	}
}

func TestRepository_CRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := repository.New[Product](db)

	if repo.Table() != "products" {
		t.Fatalf("expected table 'products', got '%s'", repo.Table())
	}

	// 1. Create
	prod := &Product{
		Name:     "Krewire Mechanical Keyboard",
		SKU:      "KREW-KEY-001",
		Price:    129.99,
		InStock:  true,
		Category: "hardware",
	}
	if err := repo.Create(ctx, prod); err != nil {
		t.Fatalf("failed to create product: %v", err)
	}
	if prod.ID <= 0 {
		t.Fatalf("expected auto-generated ID > 0, got %d", prod.ID)
	}

	// 2. FindByID
	found, err := repo.FindByID(ctx, prod.ID)
	if err != nil {
		t.Fatalf("failed to find product by ID: %v", err)
	}
	if found.Name != prod.Name || found.SKU != prod.SKU {
		t.Fatalf("found product mismatch: %+v vs %+v", found, prod)
	}

	// 3. Update
	found.Price = 149.99
	found.Name = "Krewire Mechanical Keyboard Pro"
	if err := repo.Update(ctx, found); err != nil {
		t.Fatalf("failed to update product: %v", err)
	}

	reloaded, err := repo.FindByID(ctx, prod.ID)
	if err != nil {
		t.Fatalf("failed to reload product: %v", err)
	}
	if reloaded.Price != 149.99 || reloaded.Name != "Krewire Mechanical Keyboard Pro" {
		t.Fatalf("updated fields mismatch: price=%f, name=%s", reloaded.Price, reloaded.Name)
	}

	// 4. Save (existing vs new)
	reloaded.Price = 159.99
	if err := repo.Save(ctx, reloaded); err != nil {
		t.Fatalf("failed to save existing product: %v", err)
	}

	newProd := &Product{
		Name:     "Krewire Precision Mouse",
		SKU:      "KREW-MOU-001",
		Price:    79.99,
		InStock:  true,
		Category: "hardware",
	}
	if err := repo.Save(ctx, newProd); err != nil {
		t.Fatalf("failed to save new product: %v", err)
	}
	if newProd.ID <= 0 {
		t.Fatalf("expected new product ID > 0, got %d", newProd.ID)
	}

	// 5. Delete (soft-delete since BaseModel embeds SoftDeletes)
	if err := repo.Delete(ctx, newProd.ID); err != nil {
		t.Fatalf("failed to delete product: %v", err)
	}

	// Should not be found via FindByID (soft deleted)
	deletedCheck, err := repo.FindByID(ctx, newProd.ID)
	if err == nil {
		t.Fatalf("expected error finding soft-deleted product, found %+v", deletedCheck)
	}

	// DeleteEntity
	if err := repo.DeleteEntity(ctx, reloaded); err != nil {
		t.Fatalf("failed to delete entity: %v", err)
	}
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 active products after soft deletion, got %d", count)
	}
}

func TestRepository_FindWhere_And_Count(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := repository.New[Product](db)

	prods := []*Product{
		{Name: "Book A", SKU: "BOOK-001", Price: 10.0, InStock: true, Category: "books"},
		{Name: "Book B", SKU: "BOOK-002", Price: 20.0, InStock: false, Category: "books"},
		{Name: "Laptop", SKU: "ELEC-001", Price: 1000.0, InStock: true, Category: "electronics"},
	}
	if err := repo.CreateMany(ctx, prods); err != nil {
		t.Fatalf("failed to create many: %v", err)
	}

	// FindAll
	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 items, got %d", len(all))
	}

	// FindWhere category = books
	books, err := repo.FindWhere(ctx, func(q *orm.Query) *orm.Query {
		return q.Where("category", "=", "books")
	})
	if err != nil {
		t.Fatalf("FindWhere failed: %v", err)
	}
	if len(books) != 2 {
		t.Fatalf("expected 2 books, got %d", len(books))
	}

	// FirstWhere SKU = ELEC-001
	laptop, err := repo.FirstWhere(ctx, func(q *orm.Query) *orm.Query {
		return q.Where("sku", "=", "ELEC-001")
	})
	if err != nil {
		t.Fatalf("FirstWhere failed: %v", err)
	}
	if laptop.Name != "Laptop" {
		t.Fatalf("expected Laptop, got %s", laptop.Name)
	}

	// Count & Exists
	count, err := repo.Count(ctx, func(q *orm.Query) *orm.Query {
		return q.Where("in_stock", "=", 1)
	})
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 items in stock, got %d", count)
	}

	exists, err := repo.Exists(ctx, func(q *orm.Query) *orm.Query {
		return q.Where("sku", "=", "ELEC-001")
	})
	if err != nil || !exists {
		t.Fatalf("expected exists=true, got %v (err=%v)", exists, err)
	}

	notExists, err := repo.Exists(ctx, func(q *orm.Query) *orm.Query {
		return q.Where("sku", "=", "NON-EXISTENT")
	})
	if err != nil || notExists {
		t.Fatalf("expected exists=false, got %v (err=%v)", notExists, err)
	}
}

func TestRepository_Paginate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := repository.New[Product](db)

	var items []*Product
	for i := 1; i <= 25; i++ {
		items = append(items, &Product{
			Name:     fmt.Sprintf("Item %d", i),
			SKU:      fmt.Sprintf("SKU-%03d", i),
			Price:    float64(i * 10),
			InStock:  true,
			Category: "bulk",
		})
	}
	if err := repo.CreateMany(ctx, items); err != nil {
		t.Fatalf("failed to insert 25 items: %v", err)
	}

	// Page 1 of size 10
	p1, err := repo.Paginate(ctx, 1, 10)
	if err != nil {
		t.Fatalf("paginate page 1 failed: %v", err)
	}
	if p1.Total != 25 || p1.TotalPages != 3 || len(p1.Items) != 10 {
		t.Fatalf("page 1 meta mismatch: total=%d, totalPages=%d, len=%d", p1.Total, p1.TotalPages, len(p1.Items))
	}
	if !p1.HasNext || p1.HasPrev {
		t.Fatalf("page 1 nav mismatch: hasNext=%v, hasPrev=%v", p1.HasNext, p1.HasPrev)
	}

	// Page 3 of size 10 (last page with 5 items)
	p3, err := repo.Paginate(ctx, 3, 10)
	if err != nil {
		t.Fatalf("paginate page 3 failed: %v", err)
	}
	if len(p3.Items) != 5 {
		t.Fatalf("expected 5 items on page 3, got %d", len(p3.Items))
	}
	if p3.HasNext || !p3.HasPrev {
		t.Fatalf("page 3 nav mismatch: hasNext=%v, hasPrev=%v", p3.HasNext, p3.HasPrev)
	}
}

func TestRepository_Transaction(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := repository.New[Product](db)

	// 1. Success commit
	err := repo.Transaction(ctx, func(txRepo repository.Repository[Product]) error {
		p1 := &Product{Name: "TX Prod 1", SKU: "TX-001", Price: 50.0, InStock: true, Category: "tx"}
		p2 := &Product{Name: "TX Prod 2", SKU: "TX-002", Price: 60.0, InStock: true, Category: "tx"}
		if err := txRepo.Create(ctx, p1); err != nil {
			return err
		}
		return txRepo.Create(ctx, p2)
	})
	if err != nil {
		t.Fatalf("transaction commit failed: %v", err)
	}

	count, _ := repo.Count(ctx)
	if count != 2 {
		t.Fatalf("expected 2 items after committed transaction, got %d", count)
	}

	// 2. Rollback on error
	err = repo.Transaction(ctx, func(txRepo repository.Repository[Product]) error {
		p3 := &Product{Name: "TX Prod 3", SKU: "TX-003", Price: 70.0, InStock: true, Category: "tx"}
		if err := txRepo.Create(ctx, p3); err != nil {
			return err
		}
		// deliberate error to trigger rollback
		return errors.New("simulated business logic failure")
	})
	if err == nil {
		t.Fatal("expected error from transaction rollback")
	}

	countAfterRollback, _ := repo.Count(ctx)
	if countAfterRollback != 2 {
		t.Fatalf("expected count 2 after rollback, got %d", countAfterRollback)
	}
}
