package examples

import (
	"context"
	"testing"
	"time"

	"gorm.io/cmd/gorm/examples/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// Use in-memory database with shared cache to ensure clean state
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Run migrations
	err = db.AutoMigrate(&models.User{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func TestUserQueries(t *testing.T) {
	db := setupTestDB(t)

	users := []models.User{
		{Name: "@name", Age: 28, Role: "special"}, // Add user with "@name" to test GetByID
		{Name: "user1", Age: 30, Role: "user"},
		{Name: "admin", Age: 25, Role: "admin"},
		{Name: "guest", Age: 35, Role: "guest"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("failed to seed users: %v", err)
	}

	t.Run("Test GetByID", func(t *testing.T) {
		query := Query[models.User](db)
		user, err := query.GetByID(context.Background(), int(users[0].ID)) // Get the "@name" user
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// The generated SQL has a hardcoded "@name" condition, so it should match the user with name "@name"
		if user.Name != "@name" {
			t.Errorf("expected user '@name', got: %+v", user)
		}
	})

	t.Run("Test FilterWithColumn", func(t *testing.T) {
		query := Query[models.User](db)
		u, err := query.FilterWithColumn(context.Background(), "role", "special")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if u.Role != "special" {
			t.Errorf("expected role 'special', got: %+v", u)
		}
	})

	t.Run("Test QueryWith", func(t *testing.T) {
		query := Query[models.User](db)
		result, err := query.QueryWith(context.Background(), models.User{Name: "guest"})
		if err != nil || result.Role != "guest" {
			t.Errorf("expected one 'guest', got: %+v, error: %v", result, err)
		}
	})

	t.Run("Test UpdateInfo", func(t *testing.T) {
		query := Query[models.User](db)
		update := models.User{Name: "updatedUser", Age: 40}
		if err := query.UpdateInfo(context.Background(), update, int(users[1].ID)); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Test Filter", func(t *testing.T) {
		query := Query[models.User](db)
		// The filter method requires Name != "" && Age > 0 to add conditions
		filters := []models.User{
			{Name: "admin", Age: 25, Role: "admin"},
			{Name: "guest", Age: 35, Role: "guest"},
		}
		results, err := query.Filter(context.Background(), filters)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 users, got: %d", len(results))
		}
	})

	t.Run("Test FilterByNameAndAge", func(t *testing.T) {
		query := Query[models.User](db)
		result := query.FilterByNameAndAge(context.Background(), "user1", 30)
		if result == nil {
			t.Error("expected a valid query result, got nil")
		}
	})

	t.Run("Test FilterWithTime", func(t *testing.T) {
		query := Query[models.User](db)
		start := time.Now().Add(-1 * time.Hour)
		end := time.Now().Add(1 * time.Hour)
		results, err := query.FilterWithTime(context.Background(), start, end)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(results) != len(users) {
			t.Errorf("expected %d users, got: %d", len(users), len(results))
		}
	})
}
