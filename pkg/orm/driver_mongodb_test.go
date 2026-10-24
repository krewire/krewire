package orm_test

import (
	"context"
	"testing"

	"github.com/krewire/krewire/pkg/orm"
)

type MongoUser struct {
	ID    string `json:"_id,omitempty"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func TestMongoDB_Operations(t *testing.T) {
	adapter, err := orm.NewMongoDB(orm.MongoConfig{
		URI:      "memory://",
		Database: "test_db",
	})
	if err != nil {
		t.Fatalf("NewMongoDB failed: %v", err)
	}
	defer adapter.Close()

	if adapter.Name() != "mongodb" {
		t.Errorf("expected name mongodb, got %s", adapter.Name())
	}
	if adapter.Type() != orm.AdapterTypeDocument {
		t.Errorf("expected document adapter type, got %s", adapter.Type())
	}

	ctx := context.Background()
	if err := adapter.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	col := adapter.Collection("users")

	// 1. InsertOne
	id1, err := col.InsertOne(ctx, MongoUser{
		Name:  "Budi Santoso",
		Email: "budi@example.com",
		Role:  "developer",
	})
	if err != nil {
		t.Fatalf("InsertOne failed: %v", err)
	}
	if id1 == "" {
		t.Fatal("expected non-empty document id")
	}

	id2, err := col.InsertOne(ctx, map[string]interface{}{
		"name":  "Siti Rahma",
		"email": "siti@example.com",
		"role":  "designer",
	})
	if err != nil {
		t.Fatalf("InsertOne map failed: %v", err)
	}
	if id2 == "" {
		t.Fatal("expected non-empty document id")
	}

	// 2. CountDocuments
	count, err := col.CountDocuments(ctx, nil)
	if err != nil {
		t.Fatalf("CountDocuments failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	// 3. FindOne
	var found MongoUser
	err = col.FindOne(ctx, map[string]interface{}{"email": "budi@example.com"}, &found)
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}
	if found.Name != "Budi Santoso" || found.ID != id1 {
		t.Errorf("unexpected found doc: %+v", found)
	}

	// 4. UpdateOne
	modified, err := col.UpdateOne(ctx, map[string]interface{}{"email": "budi@example.com"}, map[string]interface{}{
		"$set": map[string]interface{}{
			"role": "lead_developer",
		},
	})
	if err != nil {
		t.Fatalf("UpdateOne failed: %v", err)
	}
	if modified != 1 {
		t.Errorf("expected 1 modified document, got %d", modified)
	}

	var updated MongoUser
	_ = col.FindOne(ctx, map[string]interface{}{"_id": id1}, &updated)
	if updated.Role != "lead_developer" {
		t.Errorf("expected role lead_developer, got %s", updated.Role)
	}

	// 5. Find All into slice
	var allUsers []MongoUser
	err = col.Find(ctx, nil, &allUsers)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if len(allUsers) != 2 {
		t.Errorf("expected 2 users, got %d", len(allUsers))
	}

	// 6. DeleteOne
	deleted, err := col.DeleteOne(ctx, map[string]interface{}{"_id": id2})
	if err != nil {
		t.Fatalf("DeleteOne failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted doc, got %d", deleted)
	}

	countAfter, _ := col.CountDocuments(ctx, nil)
	if countAfter != 1 {
		t.Errorf("expected 1 doc remaining, got %d", countAfter)
	}
}
