package examples

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"gorm.io/cmd/gorm/examples/models"
	generated "gorm.io/cmd/gorm/examples/output/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// Test that generated field types match expectations at compile time.
func TestGeneratedModels_FieldTypes(t *testing.T) {
	// User
	_ = generated.User.ID
	_ = generated.User.CreatedAt
	_ = generated.User.UpdatedAt
	_ = generated.User.DeletedAt
	_ = generated.User.Name
	_ = generated.User.Age
	_ = generated.User.Birthday
	_ = generated.User.CompanyID
	_ = generated.User.ManagerID
	_ = generated.User.Role
	_ = generated.User.IsAdult

	// Account
	_ = generated.Account.ID
	_ = generated.Account.CreatedAt
	_ = generated.Account.UpdatedAt
	_ = generated.Account.DeletedAt
	_ = generated.Account.UserID
	_ = generated.Account.Number

	// Pet
	_ = generated.Pet.ID
	_ = generated.Pet.CreatedAt
	_ = generated.Pet.UpdatedAt
	_ = generated.Pet.DeletedAt
	_ = generated.Pet.UserID
	_ = generated.Pet.Name

	// Toy
	_ = generated.Toy.ID
	_ = generated.Toy.CreatedAt
	_ = generated.Toy.UpdatedAt
	_ = generated.Toy.DeletedAt
	_ = generated.Toy.Name
	_ = generated.Toy.OwnerID
	_ = generated.Toy.OwnerType

	// Company
	_ = generated.Company.ID
	_ = generated.Company.Name

	// Language
	_ = generated.Language.Code
	_ = generated.Language.Name
}

// helper to extract Column from common clause expressions
func getColumnFromExpr(expr clause.Expression) clause.Column {
	switch v := expr.(type) {
	case clause.Eq:
		return v.Column.(clause.Column)
	case clause.Neq:
		return v.Column.(clause.Column)
	case clause.Gt:
		return v.Column.(clause.Column)
	case clause.Gte:
		return v.Column.(clause.Column)
	case clause.Lt:
		return v.Column.(clause.Column)
	case clause.Lte:
		return v.Column.(clause.Column)
	case clause.IN:
		return v.Column.(clause.Column)
	case clause.Expr:
		if len(v.Vars) > 0 {
			if c, ok := v.Vars[0].(clause.Column); ok {
				return c
			}
		}
	}
	return clause.Column{}
}

func TestGeneratedModels_ColumnNames(t *testing.T) {
	if col := getColumnFromExpr(generated.User.ID.Eq(1)); col.Name != "id" {
		t.Fatalf("User.ID column = %q, want %q", col.Name, "id")
	}
	if col := getColumnFromExpr(generated.User.Name.Eq("n")); col.Name != "name" {
		t.Fatalf("User.Name column = %q, want %q", col.Name, "name")
	}
	if col := getColumnFromExpr(generated.User.Age.Gt(10)); col.Name != "age" {
		t.Fatalf("User.Age column = %q, want %q", col.Name, "age")
	}
	if col := getColumnFromExpr(generated.User.Birthday.IsNull()); col.Name != "birthday" {
		t.Fatalf("User.Birthday column = %q, want %q", col.Name, "birthday")
	}
	if col := getColumnFromExpr(generated.User.Score.IsNull()); col.Name != "score" {
		t.Fatalf("User.Score column = %q, want %q", col.Name, "score")
	}
	if col := getColumnFromExpr(generated.User.LastLogin.IsNull()); col.Name != "last_login" {
		t.Fatalf("User.LastLogin column = %q, want %q", col.Name, "last_login")
	}
	if col := getColumnFromExpr(generated.User.CompanyID.Eq(1)); col.Name != "company_id" {
		t.Fatalf("User.CompanyID column = %q, want %q", col.Name, "company_id")
	}
	if col := getColumnFromExpr(generated.User.ManagerID.Eq(1)); col.Name != "manager_id" {
		t.Fatalf("User.ManagerID column = %q, want %q", col.Name, "manager_id")
	}
	if col := getColumnFromExpr(generated.User.IsAdult.Eq(true)); col.Name != "is_adult" {
		t.Fatalf("User.IsAdult column = %q, want %q", col.Name, "is_adult")
	}
	if col := getColumnFromExpr(generated.User.DeletedAt.IsNull()); col.Name != "deleted_at" {
		t.Fatalf("User.DeletedAt column = %q, want %q", col.Name, "deleted_at")
	}

	if col := getColumnFromExpr(generated.Account.UserID.IsNull()); col.Name != "user_id" {
		t.Fatalf("Account.UserID column = %q, want %q", col.Name, "user_id")
	}
	if col := getColumnFromExpr(generated.Account.Number.Eq("n")); col.Name != "number" {
		t.Fatalf("Account.Number column = %q, want %q", col.Name, "number")
	}
	if col := getColumnFromExpr(generated.Account.RewardPoints.IsNull()); col.Name != "reward_points" {
		t.Fatalf("Account.RewardPoints column = %q, want %q", col.Name, "reward_points")
	}
	if col := getColumnFromExpr(generated.Account.LastUsedAt.IsNull()); col.Name != "last_used_at" {
		t.Fatalf("Account.LastUsedAt column = %q, want %q", col.Name, "last_used_at")
	}
	if col := getColumnFromExpr(generated.Pet.UserID.Eq(1)); col.Name != "user_id" {
		t.Fatalf("Pet.UserID column = %q, want %q", col.Name, "user_id")
	}
	if col := getColumnFromExpr(generated.Pet.Name.Eq("n")); col.Name != "name" {
		t.Fatalf("Pet.Name column = %q, want %q", col.Name, "name")
	}
	if col := getColumnFromExpr(generated.Toy.OwnerID.Eq(1)); col.Name != "owner_id" {
		t.Fatalf("Toy.OwnerID column = %q, want %q", col.Name, "owner_id")
	}
	if col := getColumnFromExpr(generated.Toy.OwnerType.Eq("t")); col.Name != "owner_type" {
		t.Fatalf("Toy.OwnerType column = %q, want %q", col.Name, "owner_type")
	}
	if col := getColumnFromExpr(generated.Company.ID.Eq(1)); col.Name != "id" {
		t.Fatalf("Company.ID column = %q, want %q", col.Name, "id")
	}
	if col := getColumnFromExpr(generated.Company.Name.Eq("n")); col.Name != "name" {
		t.Fatalf("Company.Name column = %q, want %q", col.Name, "name")
	}
	if col := getColumnFromExpr(generated.Language.Code.Eq("c")); col.Name != "code" {
		t.Fatalf("Language.Code column = %q, want %q", col.Name, "code")
	}
	if col := getColumnFromExpr(generated.Language.Name.Eq("n")); col.Name != "name" {
		t.Fatalf("Language.Name column = %q, want %q", col.Name, "name")
	}
}

func TestGeneratedModels_NoAssociationsInFields(t *testing.T) {
	// For User, ensure association fields are not generated as columns
	disallowed := []string{"Account", "Pets", "Toys", "Company", "Manager", "Team", "Languages", "Friends"}
	typ := reflect.TypeOf(generated.User)
	for _, name := range disallowed {
		if _, ok := typ.FieldByName(name); ok {
			t.Fatalf("unexpected association field %q found in generated.User", name)
		}
	}

	// For Pet, ensure Toy association is not included
	if _, ok := reflect.TypeOf(generated.Pet).FieldByName("Toy"); ok {
		t.Fatalf("unexpected association field %q found in generated.Pet", "Toy")
	}

	// For Account, ensure no back-reference association field (User) exists
	if _, ok := reflect.TypeOf(generated.Account).FieldByName("User"); ok {
		t.Fatalf("unexpected association field %q found in generated.Account", "User")
	}
}

func TestFieldHelpers_NullTypes_DBChecks(t *testing.T) {
	db := setupTestDB(t)
	// Seed default users (Score/LastLogin are NULL by default)
	seedUsers(t, db)

	ctx := context.Background()

	// Initially, all default users should have LastLogin IS NULL
	cnt, err := gorm.G[models.User](db).
		Where(generated.User.LastLogin.IsNull()).
		Count(ctx, "*")
	if err != nil {
		t.Fatalf("count LastLogin IS NULL failed: %v", err)
	}
	if cnt != 4 {
		t.Fatalf("expected 4 users with LastLogin IS NULL, got %d", cnt)
	}

	// Insert extra users with non-null values for Null fields
	now := time.Now()
	extra := []models.User{
		{Name: "eva", Age: 25, Role: "active", IsAdult: true, LastLogin: sql.NullTime{Time: now, Valid: true}},
		{Name: "frank", Age: 22, Role: "pending", IsAdult: true, Score: sql.NullInt64{Int64: 50, Valid: true}},
	}
	if err := db.Create(&extra).Error; err != nil {
		t.Fatalf("failed to insert extra users: %v", err)
	}

	// Query users with LastLogin IS NOT NULL -> expect 1 (eva)
	cntNotNull, err := gorm.G[models.User](db).
		Where(generated.User.LastLogin.IsNotNull()).
		Count(ctx, "*")
	if err != nil {
		t.Fatalf("count LastLogin IS NOT NULL failed: %v", err)
	}
	if cntNotNull != 1 {
		t.Fatalf("expected 1 user with LastLogin IS NOT NULL, got %d", cntNotNull)
	}

	// Update a seeded user to set a non-null Score
	if _, err := gorm.G[models.User](db).
		Where(generated.User.Name.Eq("bob")).
		Update(ctx, "score", sql.NullInt64{Int64: 88, Valid: true}); err != nil {
		t.Fatalf("failed to update score for bob: %v", err)
	}

	// Query users with Score IS NOT NULL -> expect 2 (frank, bob)
	cntScoreNotNull, err := gorm.G[models.User](db).
		Where(generated.User.Score.IsNotNull()).
		Count(ctx, "*")
	if err != nil {
		t.Fatalf("count Score IS NOT NULL failed: %v", err)
	}
	if cntScoreNotNull != 2 {
		t.Fatalf("expected 2 users with Score IS NOT NULL, got %d", cntScoreNotNull)
	}

	// Test Account null fields as well
	acc := models.Account{Number: "A1", RewardPoints: sql.NullInt64{Int64: 10, Valid: true}}
	if err := db.Create(&acc).Error; err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	// RewardPoints NOT NULL -> expect 1
	cntRP, err := gorm.G[models.Account](db).
		Where(generated.Account.RewardPoints.IsNotNull()).
		Count(ctx, "*")
	if err != nil {
		t.Fatalf("count RewardPoints IS NOT NULL failed: %v", err)
	}
	if cntRP != 1 {
		t.Fatalf("expected 1 account with RewardPoints IS NOT NULL, got %d", cntRP)
	}

	// Set LastUsedAt for the same account
	if _, err := gorm.G[models.Account](db).
		Where(generated.Account.Number.Eq("A1")).
		Update(ctx, "last_used_at", sql.NullTime{Time: now, Valid: true}); err != nil {
		t.Fatalf("failed to update account last_used_at: %v", err)
	}
	cntLUA, err := gorm.G[models.Account](db).
		Where(generated.Account.LastUsedAt.IsNotNull()).
		Count(ctx, "*")
	if err != nil {
		t.Fatalf("count LastUsedAt IS NOT NULL failed: %v", err)
	}
	if cntLUA != 1 {
		t.Fatalf("expected 1 account with LastUsedAt IS NOT NULL, got %d", cntLUA)
	}
}
