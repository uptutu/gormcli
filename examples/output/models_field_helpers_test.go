package examples

import (
    "context"
    "testing"

    generated "gorm.io/cmd/gorm/examples/output/models"
    "gorm.io/cmd/gorm/examples/models"
    "gorm.io/gorm"
)

func TestFieldHelpers_SingleCondition_FindWithContext(t *testing.T) {
    db := setupTestDB(t)
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
    db := setupTestDB(t)
    seedUsers(t, db)

    // Multiple conditions: age > 18 AND role = "active"
    var usersOver18Active []models.User
    if err := gorm.G[models.User](db).
        Where(generated.User.Age.Gt(18), generated.User.Role.Eq("active")).
        Scan(context.Background(), &usersOver18Active); err != nil {
        t.Fatalf("Scan(ctx, &slice) failed: %v", err)
    }
    if len(usersOver18Active) != 1 {
        t.Fatalf("expected 1 user (age>18 & active), got %d", len(usersOver18Active))
    }
}

func TestFieldHelpers_Update_UsingHelpers(t *testing.T) {
    db := setupTestDB(t)
    seedUsers(t, db)

    // Update using helper-based WHERE: set role from "pending" -> "active"
    if _, err := gorm.G[models.User](db).
        Where(generated.User.Role.Eq("pending")).
        Update(context.Background(), "role", "active"); err != nil {
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

func TestFieldHelpers_Count(t *testing.T) {
    db := setupTestDB(t)
    seedUsers(t, db)

    // Expect 2 active users from default seed
    cnt, err := gorm.G[models.User](db).
        Where(generated.User.Role.Eq("active")).
        Count(context.Background(), "*")
    if err != nil {
        t.Fatalf("count failed: %v", err)
    }
    if cnt != 2 {
        t.Fatalf("expected 2 active users, got %d", cnt)
    }
}

func TestFieldHelpers_Delete(t *testing.T) {
    db := setupTestDB(t)
    seedUsers(t, db)

    // Delete pendings
    rows, err := gorm.G[models.User](db).
        Where(generated.User.Role.Eq("pending")).
        Delete(context.Background())
    if err != nil {
        t.Fatalf("delete failed: %v", err)
    }
    if rows != 2 {
        t.Fatalf("expected to delete 2 rows, got %d", rows)
    }

    // Verify none pending left
    pending, err := gorm.G[models.User](db).
        Where(generated.User.Role.Eq("pending")).
        Find(context.Background())
    if err != nil {
        t.Fatalf("verify query failed: %v", err)
    }
    if len(pending) != 0 {
        t.Fatalf("expected 0 pending, got %d", len(pending))
    }
}
