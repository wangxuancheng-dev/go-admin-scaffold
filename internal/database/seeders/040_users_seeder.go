package seeders

import (
	"time"

	"app/internal/database/seeder"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func init() {
	Register("users", &seeder.Seeder{
		Name:         "users",
		Description:  "Create default users",
		Dependencies: []string{"roles"}, // This seeder depends on roles being created first
		Run: func(tx *gorm.DB) error {
			// Hash password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
			if err != nil {
				return err
			}

			// Create admin user
			adminUser := map[string]interface{}{
				"username":   "admin",
				"password":   string(hashedPassword),
				"email":      "admin@example.com",
				"nickname":   "Administrator",
				"status":     1,
				"created_at": time.Now(),
				"updated_at": time.Now(),
			}

			if err := tx.Table("users").Create(adminUser).Error; err != nil {
				return err
			}

			// Create test users
			testUsers := []map[string]interface{}{
				{
					"username":   "test1",
					"password":   string(hashedPassword),
					"email":      "test1@example.com",
					"nickname":   "Test User 1",
					"status":     1,
					"created_at": time.Now(),
					"updated_at": time.Now(),
				},
				{
					"username":   "test2",
					"password":   string(hashedPassword),
					"email":      "test2@example.com",
					"nickname":   "Test User 2",
					"status":     1,
					"created_at": time.Now(),
					"updated_at": time.Now(),
				},
			}

			// Batch insert test users
			return tx.Table("users").Create(testUsers).Error
		},
	})
}
