package mongodb_test

import (
	"context"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/orm"
	"github.com/krewire/krewire/pkg/orm/mongodb"
)

type Product struct {
	ID       string   `json:"_id,omitempty"`
	SKU      string   `json:"sku"`
	Name     string   `json:"name"`
	Price    float64  `json:"price"`
	Stock    int      `json:"stock"`
	Tags     []string `json:"tags"`
	Category string   `json:"category"`
	Active   bool     `json:"active"`
}

func TestMongoProductionAndAdvancedOperators(t *testing.T) {
	ctx := context.Background()
	adapter, err := mongodb.OpenProduction("memory://", "ecommerce_prod")
	if err != nil {
		t.Fatalf("OpenProduction failed: %v", err)
	}
	defer adapter.Close()

	if adapter.Database() != "ecommerce_prod" {
		t.Errorf("expected database ecommerce_prod, got %s", adapter.Database())
	}

	col := adapter.Collection("products")

	// 1. Create Unique Index on SKU
	idxName, err := col.CreateIndex(ctx, map[string]int{"sku": 1}, orm.IndexOptions{
		Name:   "sku_unique_idx",
		Unique: true,
	})
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}
	if idxName != "sku_unique_idx" {
		t.Errorf("expected idxName sku_unique_idx, got %s", idxName)
	}

	indexes, err := col.ListIndexes(ctx)
	if err != nil {
		t.Fatalf("ListIndexes failed: %v", err)
	}
	if len(indexes) != 2 { // _id_ and sku_unique_idx
		t.Errorf("expected 2 indexes, got %d", len(indexes))
	}

	// 2. Batch InsertMany
	products := []interface{}{
		Product{SKU: "LAP-001", Name: "ThinkPad X1 Carbon", Price: 1899.99, Stock: 15, Tags: []string{"laptop", "lenovo"}, Category: "computers", Active: true},
		Product{SKU: "LAP-002", Name: "MacBook Pro 16", Price: 2499.00, Stock: 8, Tags: []string{"laptop", "apple"}, Category: "computers", Active: true},
		Product{SKU: "MOU-001", Name: "MX Master 3S", Price: 99.99, Stock: 50, Tags: []string{"mouse", "logitech"}, Category: "accessories", Active: true},
		Product{SKU: "KEY-001", Name: "Keychron K2", Price: 89.00, Stock: 25, Tags: []string{"keyboard", "mechanical"}, Category: "accessories", Active: true},
		Product{SKU: "DIS-001", Name: "Dell UltraSharp 27", Price: 599.50, Stock: 0, Tags: []string{"monitor", "dell"}, Category: "displays", Active: false},
	}

	ids, err := col.InsertMany(ctx, products)
	if err != nil {
		t.Fatalf("InsertMany failed: %v", err)
	}
	if len(ids) != 5 {
		t.Fatalf("expected 5 inserted IDs, got %d", len(ids))
	}

	// 3. Test Unique Index Constraint (Duplicate Key Error)
	_, err = col.InsertOne(ctx, Product{SKU: "LAP-001", Name: "Duplicate Laptop", Price: 1200.00})
	if err == nil || !strings.Contains(err.Error(), "duplicate key error") {
		t.Fatalf("expected duplicate key error for SKU LAP-001, got: %v", err)
	}

	// 4. Comparison Operators ($gt, $lte, $in, $ne)
	var expProducts []Product
	err = col.Find(ctx, map[string]interface{}{
		"price": map[string]interface{}{"$gt": 1000.0},
	}, &expProducts)
	if err != nil {
		t.Fatalf("Find with $gt failed: %v", err)
	}
	if len(expProducts) != 2 {
		t.Errorf("expected 2 products > 1000, got %d", len(expProducts))
	}

	var inProducts []Product
	err = col.Find(ctx, map[string]interface{}{
		"category": map[string]interface{}{"$in": []string{"accessories", "displays"}},
	}, &inProducts)
	if err != nil {
		t.Fatalf("Find with $in failed: %v", err)
	}
	if len(inProducts) != 3 {
		t.Errorf("expected 3 products in accessories/displays, got %d", len(inProducts))
	}

	// 5. Logical Operators ($and, $or)
	var activeComputers []Product
	err = col.Find(ctx, map[string]interface{}{
		"$and": []map[string]interface{}{
			{"category": "computers"},
			{"active": true},
			{"price": map[string]interface{}{"$lt": 2000.0}},
		},
	}, &activeComputers)
	if err != nil {
		t.Fatalf("Find with $and failed: %v", err)
	}
	if len(activeComputers) != 1 || activeComputers[0].SKU != "LAP-001" {
		t.Errorf("unexpected active computers: %+v", activeComputers)
	}

	// 6. Regex Query ($regex)
	var thinkpads []Product
	err = col.Find(ctx, map[string]interface{}{
		"name": map[string]interface{}{"$regex": "(?i)thinkpad"},
	}, &thinkpads)
	if err != nil {
		t.Fatalf("Find with $regex failed: %v", err)
	}
	if len(thinkpads) != 1 || thinkpads[0].SKU != "LAP-001" {
		t.Errorf("expected ThinkPad match, got %+v", thinkpads)
	}

	// 7. FindWithOptions (Sort, Limit, Skip, Projection)
	var sortedAsc []Product
	err = col.FindWithOptions(ctx, map[string]interface{}{}, &sortedAsc, orm.FindOptions{
		Sort:  map[string]int{"price": 1}, // ascending price
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("FindWithOptions failed: %v", err)
	}
	if len(sortedAsc) != 2 || sortedAsc[0].SKU != "KEY-001" || sortedAsc[1].SKU != "MOU-001" {
		t.Errorf("unexpected sorted products: %+v", sortedAsc)
	}

	// 8. Updates ($inc, $push, $set, $unset)
	modified, err := col.UpdateOne(ctx, map[string]interface{}{"sku": "MOU-001"}, map[string]interface{}{
		"$inc":  map[string]interface{}{"stock": 5},
		"$push": map[string]interface{}{"tags": "ergonomic"},
	})
	if err != nil {
		t.Fatalf("UpdateOne with $inc and $push failed: %v", err)
	}
	if modified != 1 {
		t.Errorf("expected 1 modified doc, got %d", modified)
	}

	var updatedMouse Product
	_ = col.FindOne(ctx, map[string]interface{}{"sku": "MOU-001"}, &updatedMouse)
	if updatedMouse.Stock != 55 {
		t.Errorf("expected stock 55, got %d", updatedMouse.Stock)
	}
	if len(updatedMouse.Tags) != 3 || updatedMouse.Tags[2] != "ergonomic" {
		t.Errorf("expected tag ergonomic pushed, got %v", updatedMouse.Tags)
	}

	// 9. Distinct values
	cats, err := col.Distinct(ctx, "category", nil)
	if err != nil {
		t.Fatalf("Distinct failed: %v", err)
	}
	if len(cats) != 3 { // computers, accessories, displays
		t.Errorf("expected 3 distinct categories, got %v", cats)
	}

	// 10. EstimatedDocumentCount & DropDatabase
	count, err := col.EstimatedDocumentCount(ctx)
	if err != nil {
		t.Fatalf("EstimatedDocumentCount failed: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5 estimated documents, got %d", count)
	}

	if err := adapter.DropDatabase(ctx); err != nil {
		t.Fatalf("DropDatabase failed: %v", err)
	}
	afterDrop, _ := col.EstimatedDocumentCount(ctx)
	if afterDrop != 0 {
		t.Errorf("expected 0 documents after drop, got %d", afterDrop)
	}
}
