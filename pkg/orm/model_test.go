package orm_test

import (
	"context"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/orm"
	"github.com/krewire/krewire/pkg/orm/sqlite"
)

type Article struct {
	orm.BaseModel
	Title     string `orm:"column:title"`
	Slug      string `orm:"column:slug;unique"`
	Content   string `orm:"column:content"`
	ViewCount int    `orm:"column:view_count;default:0"`
	HookLog   string `orm:"-"`
}

func (Article) TableName() string {
	return "articles"
}

// Implement lifecycle hooks
func (a *Article) BeforeCreate(ctx context.Context) error {
	a.HookLog += "[BeforeCreate]"
	if a.Slug == "" {
		a.Slug = "auto-" + a.Title
	}
	return nil
}

func (a *Article) AfterCreate(ctx context.Context) error {
	a.HookLog += "[AfterCreate]"
	return nil
}

func (a *Article) BeforeUpdate(ctx context.Context) error {
	a.HookLog += "[BeforeUpdate]"
	return nil
}

func (a *Article) AfterUpdate(ctx context.Context) error {
	a.HookLog += "[AfterUpdate]"
	return nil
}

func (a *Article) BeforeDelete(ctx context.Context) error {
	a.HookLog += "[BeforeDelete]"
	return nil
}

func (a *Article) AfterDelete(ctx context.Context) error {
	a.HookLog += "[AfterDelete]"
	return nil
}

func TestModelLifecycleHooksAndTimestamps(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if err := db.AutoMigrate(ctx, &Article{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	art := &Article{
		Title:   "Introducing Krewire",
		Content: "A modern SDLC framework for Go.",
	}

	// 1. Insert & BeforeCreate / AfterCreate hooks + CreatedAt/UpdatedAt
	if _, err := db.Table("articles").Insert(art); err != nil {
		t.Fatalf("insert article failed: %v", err)
	}

	if art.ID <= 0 {
		t.Errorf("expected auto-increment ID assigned, got %d", art.ID)
	}
	if art.CreatedAt.IsZero() {
		t.Error("expected CreatedAt timestamp populated, got zero")
	}
	if art.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt timestamp populated, got zero")
	}
	if art.Slug != "auto-Introducing Krewire" {
		t.Errorf("expected BeforeCreate hook to generate slug, got %q", art.Slug)
	}
	if art.HookLog != "[BeforeCreate][AfterCreate]" {
		t.Errorf("expected hook log '[BeforeCreate][AfterCreate]', got %q", art.HookLog)
	}

	// 2. Update & BeforeUpdate / AfterUpdate hooks
	art.Title = "Introducing Krewire 2.0"
	time.Sleep(10 * time.Millisecond) // slight delay to ensure UpdatedAt changes
	oldUpdated := art.UpdatedAt

	if _, err := db.Table("articles").Update(art); err != nil {
		t.Fatalf("update article failed: %v", err)
	}

	if !art.UpdatedAt.After(oldUpdated) && !art.UpdatedAt.Equal(oldUpdated) {
		t.Errorf("expected UpdatedAt to update, got %v", art.UpdatedAt)
	}
	if art.HookLog != "[BeforeCreate][AfterCreate][BeforeUpdate][AfterUpdate]" {
		t.Errorf("expected update hooks executed, got %q", art.HookLog)
	}
}

func TestModelSoftDeletesAndQueryExtensions(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if err := db.AutoMigrate(ctx, &Article{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	a1 := &Article{Title: "Post One", Content: "Content 1"}
	a2 := &Article{Title: "Post Two", Content: "Content 2"}
	a3 := &Article{Title: "Post Three", Content: "Content 3"}

	_, _ = db.Table("articles").Insert(a1)
	_, _ = db.Table("articles").Insert(a2)
	_, _ = db.Table("articles").Insert(a3)

	// Check total active
	var allActive []Article
	if err := db.Model(&Article{}).FindAll(&allActive); err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(allActive) != 3 {
		t.Fatalf("expected 3 active articles, got %d", len(allActive))
	}

	// 1. Soft Delete a2
	if _, err := db.Model(&Article{}).Delete(a2); err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}
	if a2.DeletedAt == nil {
		t.Error("expected in-memory a2.DeletedAt populated after soft delete")
	}

	// 2. FindAll should return 2 active articles now (a2 excluded)
	var remaining []Article
	if err := db.Model(&Article{}).FindAll(&remaining); err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 active articles after soft delete, got %d", len(remaining))
	}

	// 3. WithTrashed returns 3
	var withTrash []Article
	if err := db.Model(&Article{}).WithTrashed().FindAll(&withTrash); err != nil {
		t.Fatalf("WithTrashed FindAll failed: %v", err)
	}
	if len(withTrash) != 3 {
		t.Fatalf("expected 3 articles with trashed, got %d", len(withTrash))
	}

	// 4. OnlyTrashed returns 1 (a2)
	var onlyTrash []Article
	if err := db.Model(&Article{}).OnlyTrashed().FindAll(&onlyTrash); err != nil {
		t.Fatalf("OnlyTrashed FindAll failed: %v", err)
	}
	if len(onlyTrash) != 1 || onlyTrash[0].Title != "Post Two" {
		t.Fatalf("expected 1 trashed article (Post Two), got %+v", onlyTrash)
	}

	// 5. Restore a2
	if _, err := db.Model(&Article{}).Restore(a2); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	if a2.DeletedAt != nil {
		t.Error("expected a2.DeletedAt to be nil after restore")
	}

	// Verify all 3 active again
	activeCount, _ := db.Model(&Article{}).Count()
	if activeCount != 3 {
		t.Errorf("expected 3 active articles after restore, got %d", activeCount)
	}

	// 6. ForceDelete permanently deletes a3
	if _, err := db.Model(&Article{}).ForceDelete(a3); err != nil {
		t.Fatalf("ForceDelete failed: %v", err)
	}

	// Verify withTrashed now only sees 2 articles
	totalCount, _ := db.Model(&Article{}).WithTrashed().Count()
	if totalCount != 2 {
		t.Errorf("expected 2 total articles after permanent force delete, got %d", totalCount)
	}

	// 7. Test Exists & DoesntExist
	ex, err := db.Model(&Article{}).Where("title", "=", "Post One").Exists()
	if err != nil || !ex {
		t.Errorf("expected Post One to exist, got %v (err: %v)", ex, err)
	}

	dNotExist, err := db.Model(&Article{}).Where("title", "=", "NonExistent").DoesntExist()
	if err != nil || !dNotExist {
		t.Errorf("expected NonExistent to not exist, got %v (err: %v)", dNotExist, err)
	}

	// 8. Test FirstOrFail and FindOrFail
	var found Article
	if err := db.Model(&Article{}).Where("title", "=", "Post One").FirstOrFail(&found); err != nil {
		t.Fatalf("FirstOrFail failed: %v", err)
	}
	if found.Title != "Post One" {
		t.Errorf("expected Post One, got %q", found.Title)
	}

	var missing Article
	if err := db.Model(&Article{}).FindOrFail(&missing, 99999); err == nil {
		t.Error("expected FindOrFail for missing ID to return error, got nil")
	}

	// 9. Test Pluck
	var titles []string
	if err := db.Model(&Article{}).OrderBy("id", "ASC").Pluck("title", &titles); err != nil {
		t.Fatalf("Pluck failed: %v", err)
	}
	if len(titles) != 2 || titles[0] != "Post One" || titles[1] != "Post Two" {
		t.Errorf("unexpected plucked titles: %v", titles)
	}

	// 10. Test Scope
	postOneScope := func(q *orm.Query) *orm.Query {
		return q.Where("title", "=", "Post One")
	}
	var scoped Article
	if err := db.Model(&Article{}).Scope(postOneScope).Find(&scoped); err != nil {
		t.Fatalf("scoped query failed: %v", err)
	}
	if scoped.Title != "Post One" {
		t.Errorf("expected scoped query to find Post One, got %q", scoped.Title)
	}
}
