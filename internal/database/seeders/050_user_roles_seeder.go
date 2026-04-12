package seeders

import (
	"app/internal/database/seeder"

	"gorm.io/gorm"
)

func init() {
	Register("user_roles", &seeder.Seeder{
		Name:         "user_roles",
		Description:  "Create user role associations",
		Dependencies: []string{"users", "roles"},
		Run: func(tx *gorm.DB) error {
			// Get admin user ID
			var adminUser struct{ ID uint }
			if err := tx.Table("users").Where("username = ?", "admin").First(&adminUser).Error; err != nil {
				return err
			}

			// Get admin role ID
			var adminRole struct{ ID uint }
			if err := tx.Table("roles").Where("code = ?", "admin").First(&adminRole).Error; err != nil {
				return err
			}

			// Get user role ID
			var userRole struct{ ID uint }
			if err := tx.Table("roles").Where("code = ?", "user").First(&userRole).Error; err != nil {
				return err
			}

			// Get test users IDs
			var testUsers []struct{ ID uint }
			if err := tx.Table("users").Where("username LIKE ?", "test%").Find(&testUsers).Error; err != nil {
				return err
			}

			// Create user role associations
			userRoles := []map[string]interface{}{
				{
					"user_id": adminUser.ID,
					"role_id": adminRole.ID,
				},
			}

			// Add user role for test users
			for _, user := range testUsers {
				userRoles = append(userRoles, map[string]interface{}{
					"user_id": user.ID,
					"role_id": userRole.ID,
				})
			}

			return tx.Table("user_roles").Create(userRoles).Error
		},
	})
}
