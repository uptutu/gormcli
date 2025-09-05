package examples

import (
    "context"
    "testing"

    generated "gorm.io/cmd/gorm/examples/output/models"
    "gorm.io/cmd/gorm/examples/models"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func setupHelpersDB(t *testing.T) *gorm.DB {
    t.Helper()
    db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to connect database: %v", err)
    }
    if err := db.AutoMigrate(&models.User{}); err != nil {
        t.Fatalf("failed to migrate database: %v", err)
    }
    return db
}

func seedUsers(t *testing.T, db *gorm.DB) []models.User {
    t.Helper()
    users := []models.User{
        {Name: "alice", Age: 20, Role: "active", IsAdult: true},
        {Name: "bob", Age: 17, Role: "active", IsAdult: false},
        {Name: "cathy", Age: 30, Role: "pending", IsAdult: true},
        {Name: "dan", Age: 40, Role: "pending", IsAdult: true},
    }
    if err := db.Create(&users).Error; err != nil {
        t.Fatalf("failed to seed users: %v", err)
    }
    return users
}

func TestFieldHelpers_SingleCondition_FindWithContext(t *testing.T) {
    db := setupHelpersDB(t)
    seedUsers(t, db)

    ctx := context.Background()
    // Simple filter using generated field helper (Role = "active")
    got, err := gorm.G[models.User](db).
        Where(generated.User.Role.Eq("active")).
        Find(ctx)
    if err != nil {
        t.Fatalf("Find(ctx) failed: %v", err)
    }
    if len(got) != 2 {
        t.Fatalf("expected 2 active users, got %d", len(got))
    }
}

func TestFieldHelpers_MultipleConditions_FindIntoSlice(t *testing.T) {
    db := setupHelpersDB(t)
    seedUsers(t, db)

    // Multiple conditions: age > 18 AND role = "active"
    var usersOver18Active []models.User
    if err := gorm.G[models.User](db).
        Where(generated.User.Age.Gt(18), generated.User.Role.Eq("active")).
        Find(&usersOver18Active); err != nil {
        t.Fatalf("Find(&slice) failed: %v", err)
    }
    if len(usersOver18Active) != 1 {
        t.Fatalf("expected 1 user (age>18 & active), got %d", len(usersOver18Active))
    }
}

func TestFieldHelpers_Update_UsingHelpers(t *testing.T) {
    db := setupHelpersDB(t)
    seedUsers(t, db)

    // Update using helper-based WHERE: set role from "pending" -> "active"
    if err := gorm.G[models.User](db).
        Where(generated.User.Role.Eq("pending")).
        Update("role", "active"); err != nil {
        t.Fatalf("Update using helpers failed: %v", err)
    }

    // Verify: no more pending
    pending, err := gorm.G[models.User](db).
        Where(generated.User.Role.Eq("pending")).
        Find(context.Background())
    if err != nil {
        t.Fatalf("verify pending query failed: %v", err)
    }
    if len(pending) != 0 {
        t.Fatalf("expected 0 pending after update, got %d", len(pending))
    }

    // And active count increased to 4
    active, err := gorm.G[models.User](db).
        Where(generated.User.Role.Eq("active")).
        Find(context.Background())
    if err != nil {
        t.Fatalf("verify active query failed: %v", err)
    }
    if len(active) != 4 {
        t.Fatalf("expected 4 active after update, got %d", len(active))
    }
}

