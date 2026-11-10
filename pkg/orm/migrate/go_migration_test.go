package migrate_test

import (
	"context"
	"testing"

	"github.com/krewire/krewire/pkg/orm/migrate"
	"github.com/krewire/krewire/pkg/orm/schema"
	"github.com/krewire/krewire/pkg/orm/sqlite"
)

func TestGoMigrations_UpAndDown(t *testing.T) {
	migrate.ClearRegistry()

	// Register 2 migrations (like Laravel)
	migrate.Register(&migrate.GoMigration{
		Version: "20261110000001",
		Name:    "create_users_table",
		Up: func(ctx context.Context, s *schema.Builder) error {
			return s.Create("users", func(table *schema.Blueprint) {
				table.ID()
				table.String("name", 100).NotNull()
				table.String("email", 150).Unique()
				table.Timestamps()
			})
		},
		Down: func(ctx context.Context, s *schema.Builder) error {
			return s.DropIfExists("users")
		},
	})

	migrate.Register(&migrate.GoMigration{
		Version: "20261110000002",
		Name:    "create_posts_table",
		Up: func(ctx context.Context, s *schema.Builder) error {
			return s.Create("posts", func(table *schema.Blueprint) {
				table.ID()
				table.BigInteger("user_id").Index()
				table.String("title", 200).NotNull()
				table.Text("content").Nullable()
				table.Timestamps()
			})
		},
		Down: func(ctx context.Context, s *schema.Builder) error {
			return s.DropIfExists("posts")
		},
	})

	ctx := context.Background()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("sqlite.Open failed: %v", err)
	}
	defer db.Close()

	gm, err := migrate.NewGoMigrator(db, migrate.Options{})
	if err != nil {
		t.Fatalf("NewGoMigrator failed: %v", err)
	}

	// 1. Initial Status: both pending
	st, err := gm.Status(ctx)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if len(st) != 2 || st[0].State != migrate.StatePending || st[1].State != migrate.StatePending {
		t.Fatalf("unexpected initial status: %+v", st)
	}

	// 2. Up 1 step
	results1, err := gm.Up(ctx, 1)
	if err != nil {
		t.Fatalf("Up(1) failed: %v", err)
	}
	if len(results1) != 1 || results1[0].Migration.Name != "create_users_table" {
		t.Errorf("unexpected up(1) result: %+v", results1)
	}

	s := schema.NewBuilder(ctx, db)
	hasUsers, _ := s.HasTable("users")
	if !hasUsers {
		t.Error("expected users table to exist after up(1)")
	}
	hasPosts, _ := s.HasTable("posts")
	if hasPosts {
		t.Error("posts table should not exist after up(1)")
	}

	// 3. Up remaining
	results2, err := gm.Up(ctx, 0)
	if err != nil {
		t.Fatalf("Up(0) failed: %v", err)
	}
	if len(results2) != 1 || results2[0].Migration.Name != "create_posts_table" {
		t.Errorf("unexpected up(0) result: %+v", results2)
	}
	hasPosts, _ = s.HasTable("posts")
	if !hasPosts {
		t.Error("expected posts table to exist after full up")
	}

	// 4. Status: both applied
	stAfter, _ := gm.Status(ctx)
	if len(stAfter) != 2 || stAfter[0].State != migrate.StateApplied || stAfter[1].State != migrate.StateApplied {
		t.Errorf("expected both applied, got: %+v", stAfter)
	}

	// 5. Down 1 step
	downResults, err := gm.Down(ctx, 1)
	if err != nil {
		t.Fatalf("Down(1) failed: %v", err)
	}
	if len(downResults) != 1 || downResults[0].Migration.Name != "create_posts_table" {
		t.Errorf("unexpected down(1) result: %+v", downResults)
	}
	hasPostsAfterDown, _ := s.HasTable("posts")
	if hasPostsAfterDown {
		t.Error("posts table should be dropped after down(1)")
	}
	hasUsersAfterDown, _ := s.HasTable("users")
	if !hasUsersAfterDown {
		t.Error("users table should still exist after down(1)")
	}

	// 6. Redo on the remaining applied migration (users)
	redoResults, err := gm.Redo(ctx)
	if err != nil {
		t.Fatalf("Redo failed: %v", err)
	}
	if len(redoResults) != 2 || redoResults[0].Direction != migrate.DirectionDown || redoResults[1].Direction != migrate.DirectionUp {
		t.Errorf("unexpected redo results: %+v", redoResults)
	}
	hasUsersAfterRedo, _ := s.HasTable("users")
	if !hasUsersAfterRedo {
		t.Error("users table should exist after redo")
	}
}
