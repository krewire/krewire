package seeder_test

import (
	"context"
	"testing"

	"github.com/krewire/krewire/pkg/orm"
	"github.com/krewire/krewire/pkg/orm/faker"
	"github.com/krewire/krewire/pkg/orm/seeder"
	"github.com/krewire/krewire/pkg/orm/sqlite"
)

type TestUser struct {
	ID      int64  `orm:"pk;auto"`
	Name    string `orm:"column:name"`
	Email   string `orm:"column:email;unique"`
	IsAdmin bool   `orm:"column:is_admin"`
	City    string `orm:"column:city"`
}

func (TestUser) TableName() string {
	return "test_users"
}

func TestSeederAndFactory(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("sqlite open: %v", err)
	}
	defer db.Close()

	if err := db.AutoMigrate(ctx, &TestUser{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	fk := faker.New(12345)

	userFactory := seeder.NewFactory(func(f *faker.Faker) *TestUser {
		return &TestUser{
			Name:    f.Name(),
			Email:   f.Email(),
			IsAdmin: false,
			City:    f.City(),
		}
	}).WithFaker(fk)

	// 1. Test Make & State (in memory)
	adminState := func(u *TestUser) {
		u.IsAdmin = true
	}
	adminUser := userFactory.State(adminState).Make()
	if !adminUser.IsAdmin {
		t.Errorf("expected IsAdmin = true from state, got false")
	}
	if adminUser.Name == "" || adminUser.Email == "" {
		t.Errorf("expected valid generated name/email, got: %+v", adminUser)
	}

	// 2. Test Create (persisted)
	createdUser, err := userFactory.Create(ctx, db)
	if err != nil {
		t.Fatalf("userFactory.Create failed: %v", err)
	}
	if createdUser.ID <= 0 {
		t.Errorf("expected generated primary key ID > 0, got %d", createdUser.ID)
	}

	// 3. Test CreateMany
	createdMany, err := userFactory.Count(5).CreateMany(ctx, db)
	if err != nil {
		t.Fatalf("userFactory.CreateMany failed: %v", err)
	}
	if len(createdMany) != 5 {
		t.Fatalf("expected 5 created users, got %d", len(createdMany))
	}
	for i, u := range createdMany {
		if u.ID <= 0 {
			t.Errorf("expected user [%d] to have ID > 0, got %d", i, u.ID)
		}
	}

	// Verify total count in DB: 1 + 5 = 6
	count, err := db.Table("test_users").Count()
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 6 {
		t.Errorf("expected 6 users in table, got %d", count)
	}
}

func TestSeederRegistryAndCall(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("sqlite open: %v", err)
	}
	defer db.Close()

	if err := db.AutoMigrate(ctx, &TestUser{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	seeder.ClearRegistry()
	defer seeder.ClearRegistry()

	ranUserSeeder := false
	userSeeder := seeder.FuncSeeder(func(ctx context.Context, db *orm.DB) error {
		ranUserSeeder = true
		_, err := db.Table("test_users").Insert(&TestUser{
			Name:  "Seeded Admin",
			Email: "admin@krewire.com",
		})
		return err
	})

	seeder.Register("UserSeeder", userSeeder)

	// Call directly
	if err := seeder.Call(ctx, db, userSeeder); err != nil {
		t.Fatalf("seeder.Call failed: %v", err)
	}
	if !ranUserSeeder {
		t.Error("expected userSeeder to have run")
	}

	// Run via registry name
	ranSecond := false
	secondSeeder := seeder.FuncSeeder(func(ctx context.Context, db *orm.DB) error {
		ranSecond = true
		return nil
	})
	seeder.Register("SecondSeeder", secondSeeder)

	if err := seeder.Run(ctx, db, "SecondSeeder"); err != nil {
		t.Fatalf("seeder.Run failed: %v", err)
	}
	if !ranSecond {
		t.Error("expected SecondSeeder to have run")
	}

	// Test missing seeder error
	if err := seeder.Run(ctx, db, "NonExistentSeeder"); err == nil {
		t.Error("expected error for non-existent seeder, got nil")
	}
}
