package orm_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/orm"
	"github.com/krewire/krewire/pkg/orm/sqlite"
)

type RelUser struct {
	orm.BaseModel
	Name    string      `orm:"column:name"`
	Email   string      `orm:"column:email;unique"`
	Profile *RelProfile `orm:"has_one;foreign_key:user_id"`
	Posts   []RelPost   `orm:"has_many;foreign_key:user_id"`
}

func (RelUser) TableName() string { return "rel_users" }

type RelProfile struct {
	orm.BaseModel
	UserID  int64    `orm:"column:user_id;index"`
	Bio     string   `orm:"column:bio"`
	Website string   `orm:"column:website"`
	User    *RelUser `orm:"belongs_to;foreign_key:user_id"`
}

func (RelProfile) TableName() string { return "rel_profiles" }

type RelPost struct {
	orm.BaseModel
	UserID    int64    `orm:"column:user_id;index"`
	Title     string   `orm:"column:title"`
	Published bool     `orm:"column:published"`
	Author    *RelUser `orm:"belongs_to;foreign_key:user_id"`
	Tags      []RelTag `orm:"many_to_many;join_table:rel_post_tags;join_fk:post_id;join_ref:tag_id"`
}

func (RelPost) TableName() string { return "rel_posts" }

type RelTag struct {
	orm.BaseModel
	Name string `orm:"column:name;unique"`
}

func (RelTag) TableName() string { return "rel_tags" }

type RelPostTag struct {
	orm.BaseModel
	PostID int64 `orm:"column:post_id;index"`
	TagID  int64 `orm:"column:tag_id;index"`
}

func (RelPostTag) TableName() string { return "rel_post_tags" }

func setupRelDB(t *testing.T) (*orm.DB, context.Context) {
	ctx := context.Background()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite in-memory: %v", err)
	}

	if err := db.AutoMigrate(ctx, &RelUser{}, &RelProfile{}, &RelPost{}, &RelTag{}, &RelPostTag{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	return db, ctx
}

func TestModel_RelationshipParsing(t *testing.T) {
	tmUser, err := orm.ParseModel(&RelUser{})
	if err != nil {
		t.Fatalf("parse RelUser: %v", err)
	}

	if len(tmUser.Relationships) != 2 {
		t.Fatalf("expected 2 relationships on RelUser, got %d", len(tmUser.Relationships))
	}
	if tmUser.RelationshipMap["profile"] == nil || tmUser.RelationshipMap["profile"].Type != orm.RelationHasOne {
		t.Errorf("expected Profile to be has_one relation")
	}
	if tmUser.RelationshipMap["posts"] == nil || tmUser.RelationshipMap["posts"].Type != orm.RelationHasMany {
		t.Errorf("expected Posts to be has_many relation")
	}

	// Columns must not include association fields
	if _, ok := tmUser.ColumnMap["profile"]; ok {
		t.Errorf("profile should not be in ColumnMap")
	}
	if _, ok := tmUser.ColumnMap["posts"]; ok {
		t.Errorf("posts should not be in ColumnMap")
	}

	tmPost, err := orm.ParseModel(&RelPost{})
	if err != nil {
		t.Fatalf("parse RelPost: %v", err)
	}
	if tmPost.RelationshipMap["author"] == nil || tmPost.RelationshipMap["author"].Type != orm.RelationBelongsTo {
		t.Errorf("expected Author to be belongs_to relation")
	}
	if tmPost.RelationshipMap["tags"] == nil || tmPost.RelationshipMap["tags"].Type != orm.RelationManyToMany {
		t.Errorf("expected Tags to be many_to_many relation")
	}
	relTags := tmPost.RelationshipMap["tags"]
	if relTags.JoinTable != "rel_post_tags" || relTags.JoinForeignKey != "post_id" || relTags.JoinReferences != "tag_id" {
		t.Errorf("tags join metadata incorrect: %+v", relTags)
	}
}

func TestPreload_HasOne(t *testing.T) {
	db, ctx := setupRelDB(t)
	defer db.Close()

	u1 := &RelUser{Name: "Alice", Email: "alice@krewire.dev"}
	u2 := &RelUser{Name: "Bob", Email: "bob@krewire.dev"}
	if _, err := db.Table("rel_users").Insert(u1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Table("rel_users").Insert(u2); err != nil {
		t.Fatal(err)
	}

	p1 := &RelProfile{UserID: u1.ID, Bio: "Alice Bio", Website: "https://alice.dev"}
	p2 := &RelProfile{UserID: u2.ID, Bio: "Bob Bio", Website: "https://bob.dev"}
	if _, err := db.Table("rel_profiles").Insert(p1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Table("rel_profiles").Insert(p2); err != nil {
		t.Fatal(err)
	}

	// Single record Find with Preload
	var found RelUser
	if err := db.Model(&RelUser{}).Preload("Profile").Find(&found, u1.ID); err != nil {
		t.Fatalf("find with preload failed: %v", err)
	}
	if found.Profile == nil {
		t.Fatal("expected Profile to be loaded, got nil")
	}
	if found.Profile.Bio != "Alice Bio" {
		t.Errorf("expected bio 'Alice Bio', got %q", found.Profile.Bio)
	}

	// Slice FindAll with Preload
	var users []RelUser
	if err := db.Model(&RelUser{}).OrderBy("id", "asc").Preload("Profile").FindAll(&users); err != nil {
		t.Fatalf("findAll with preload failed: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Profile == nil || users[0].Profile.Bio != "Alice Bio" {
		t.Errorf("user 0 profile mismatch: %+v", users[0].Profile)
	}
	if users[1].Profile == nil || users[1].Profile.Bio != "Bob Bio" {
		t.Errorf("user 1 profile mismatch: %+v", users[1].Profile)
	}
	_ = ctx
}

func TestPreload_HasManyAndScopedPreload(t *testing.T) {
	db, _ := setupRelDB(t)
	defer db.Close()

	u := &RelUser{Name: "Charlie", Email: "charlie@krewire.dev"}
	if _, err := db.Table("rel_users").Insert(u); err != nil {
		t.Fatal(err)
	}

	posts := []RelPost{
		{UserID: u.ID, Title: "Post 1", Published: true},
		{UserID: u.ID, Title: "Post 2 Draft", Published: false},
		{UserID: u.ID, Title: "Post 3", Published: true},
	}
	for i := range posts {
		if _, err := db.Table("rel_posts").Insert(&posts[i]); err != nil {
			t.Fatal(err)
		}
	}

	// 1. All posts preloaded
	var userAll RelUser
	if err := db.Model(&RelUser{}).Preload("Posts").Find(&userAll, u.ID); err != nil {
		t.Fatalf("preload posts: %v", err)
	}
	if len(userAll.Posts) != 3 {
		t.Fatalf("expected 3 posts, got %d", len(userAll.Posts))
	}

	// 2. Scoped preload: only published
	var userPub RelUser
	err := db.Model(&RelUser{}).Preload("Posts", func(q *orm.Query) *orm.Query {
		return q.Where("published", "=", true)
	}).Find(&userPub, u.ID)
	if err != nil {
		t.Fatalf("scoped preload: %v", err)
	}
	if len(userPub.Posts) != 2 {
		t.Fatalf("expected 2 published posts, got %d", len(userPub.Posts))
	}
	for _, p := range userPub.Posts {
		if !p.Published {
			t.Errorf("expected published post only, got unpublished: %s", p.Title)
		}
	}
}

func TestPreload_BelongsTo(t *testing.T) {
	db, _ := setupRelDB(t)
	defer db.Close()

	u := &RelUser{Name: "Diana", Email: "diana@krewire.dev"}
	if _, err := db.Table("rel_users").Insert(u); err != nil {
		t.Fatal(err)
	}

	post := &RelPost{UserID: u.ID, Title: "Diana's First Post", Published: true}
	if _, err := db.Table("rel_posts").Insert(post); err != nil {
		t.Fatal(err)
	}

	var foundPost RelPost
	if err := db.Model(&RelPost{}).Preload("Author").Find(&foundPost, post.ID); err != nil {
		t.Fatalf("find post with belongs_to author: %v", err)
	}
	if foundPost.Author == nil {
		t.Fatal("expected Author to be loaded, got nil")
	}
	if foundPost.Author.Name != "Diana" {
		t.Errorf("expected author Diana, got %s", foundPost.Author.Name)
	}
}

func TestPreload_ManyToMany(t *testing.T) {
	db, _ := setupRelDB(t)
	defer db.Close()

	u := &RelUser{Name: "Eve", Email: "eve@krewire.dev"}
	if _, err := db.Table("rel_users").Insert(u); err != nil {
		t.Fatal(err)
	}

	post := &RelPost{UserID: u.ID, Title: "Krewire Deep Dive", Published: true}
	if _, err := db.Table("rel_posts").Insert(post); err != nil {
		t.Fatal(err)
	}

	tagGo := &RelTag{Name: "Golang"}
	tagKrew := &RelTag{Name: "Krewire"}
	tagTUI := &RelTag{Name: "TUI"}
	for _, tag := range []*RelTag{tagGo, tagKrew, tagTUI} {
		if _, err := db.Table("rel_tags").Insert(tag); err != nil {
			t.Fatal(err)
		}
	}

	// Link post to Go and Krewire
	for _, tagID := range []int64{tagGo.ID, tagKrew.ID} {
		pt := &RelPostTag{PostID: post.ID, TagID: tagID}
		if _, err := db.Table("rel_post_tags").Insert(pt); err != nil {
			t.Fatal(err)
		}
	}

	var loadedPost RelPost
	if err := db.Model(&RelPost{}).Preload("Tags").Find(&loadedPost, post.ID); err != nil {
		t.Fatalf("preload many_to_many tags: %v", err)
	}
	if len(loadedPost.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(loadedPost.Tags))
	}

	tagNames := map[string]bool{}
	for _, tg := range loadedPost.Tags {
		tagNames[tg.Name] = true
	}
	if !tagNames["Golang"] || !tagNames["Krewire"] {
		t.Errorf("expected Golang and Krewire tags, got %v", tagNames)
	}
}

func TestBatch_InsertMany(t *testing.T) {
	db, _ := setupRelDB(t)
	defer db.Close()

	tags := []RelTag{
		{Name: "Cloud"},
		{Name: "Auth"},
		{Name: "Dash"},
		{Name: "Internal"},
	}

	affected, err := db.Model(&RelTag{}).InsertMany(tags)
	if err != nil {
		t.Fatalf("InsertMany failed: %v", err)
	}
	if affected != 4 {
		t.Errorf("expected 4 rows affected, got %d", affected)
	}

	var allTags []RelTag
	if err := db.Model(&RelTag{}).OrderBy("name", "asc").FindAll(&allTags); err != nil {
		t.Fatal(err)
	}
	if len(allTags) != 4 {
		t.Fatalf("expected 4 tags in database, got %d", len(allTags))
	}
	for _, tg := range allTags {
		if tg.CreatedAt.IsZero() || tg.UpdatedAt.IsZero() {
			t.Errorf("tag %s has unpopulated timestamps", tg.Name)
		}
	}
}

func TestBatch_Upsert(t *testing.T) {
	db, _ := setupRelDB(t)
	defer db.Close()

	tag := &RelTag{Name: "Initial"}
	if _, err := db.Table("rel_tags").Insert(tag); err != nil {
		t.Fatal(err)
	}

	// Perform Upsert on ID conflict
	updateTag := &RelTag{
		BaseModel: orm.BaseModel{ID: tag.ID},
		Name:      "Updated Via Upsert",
	}
	affected, err := db.Model(&RelTag{}).Upsert(updateTag, []string{"id"}, []string{"name"})
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	if affected == 0 {
		t.Errorf("expected rows affected > 0, got %d", affected)
	}

	var reloaded RelTag
	if err := db.Model(&RelTag{}).Find(&reloaded, tag.ID); err != nil {
		t.Fatal(err)
	}
	if reloaded.Name != "Updated Via Upsert" {
		t.Errorf("expected 'Updated Via Upsert', got %q", reloaded.Name)
	}
}

func TestRawQuery_Operations(t *testing.T) {
	db, ctx := setupRelDB(t)
	defer db.Close()

	// 1. Raw Exec
	res, err := db.Raw("INSERT INTO rel_users (name, email, created_at, updated_at) VALUES (?, ?, ?, ?)",
		"Raw User", "raw@krewire.dev", time.Now(), time.Now()).Exec()
	if err != nil {
		t.Fatalf("raw exec failed: %v", err)
	}
	ra, _ := res.RowsAffected()
	if ra != 1 {
		t.Errorf("expected 1 row affected, got %d", ra)
	}

	// 2. Raw Scan into Struct
	var user RelUser
	if err := db.Raw("SELECT id, name, email, created_at, updated_at FROM rel_users WHERE email = ?", "raw@krewire.dev").Scan(&user); err != nil {
		t.Fatalf("raw scan struct: %v", err)
	}
	if user.Name != "Raw User" {
		t.Errorf("expected 'Raw User', got %q", user.Name)
	}

	// 3. Raw Scan into Scalar Primitive
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM rel_users").Scan(&count); err != nil {
		t.Fatalf("raw scan scalar: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// 4. Raw Scan into Slice of Structs
	var users []RelUser
	if err := db.Raw("SELECT id, name, email, created_at, updated_at FROM rel_users ORDER BY id ASC").Scan(&users); err != nil {
		t.Fatalf("raw scan slice: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}

	// 5. Raw Query in Transaction
	err = db.Transaction(ctx, func(tx *orm.Tx) error {
		_, txErr := tx.Raw("INSERT INTO rel_users (name, email, created_at, updated_at) VALUES (?, ?, ?, ?)",
			"Tx User", "tx@krewire.dev", time.Now(), time.Now()).Exec()
		return txErr
	})
	if err != nil {
		t.Fatalf("raw query in transaction: %v", err)
	}

	var total int64
	_ = db.Raw("SELECT COUNT(*) FROM rel_users").Scan(&total)
	if total != 2 {
		t.Errorf("expected 2 users after tx, got %d", total)
	}
}

func TestDB_StatsAndHealth(t *testing.T) {
	db, ctx := setupRelDB(t)
	defer db.Close()

	stats := db.Stats()
	if stats.MaxOpenConnections <= 0 {
		t.Errorf("expected MaxOpenConnections > 0, got %d", stats.MaxOpenConnections)
	}

	if err := db.Health(ctx); err != nil {
		t.Errorf("health check failed: %v", err)
	}

	// Closed DB health failure
	closedDB, _ := sqlite.Open(":memory:")
	_ = closedDB.Close()
	if err := closedDB.Health(ctx); err == nil {
		t.Error("expected health check on closed DB to return error, got nil")
	}
}

func TestDB_LoggerAndSlowQuery(t *testing.T) {
	db, _ := setupRelDB(t)
	defer db.Close()

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	db.SetLogger(logger)
	db.SetSlowQueryThreshold(500 * time.Millisecond)

	count, err := db.Model(&RelUser{}).Count()
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("orm: query executed")) {
		t.Errorf("expected log output to contain 'orm: query executed', got: %s", output)
	}
	_ = count
}
