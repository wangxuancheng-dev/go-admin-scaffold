package seeders

import (
	"app/internal/database/seeder"

	"gorm.io/gorm"
)

func init() {
	Register("role_menus", &seeder.Seeder{
		Name:         "role_menus",
		Description:  "Create role-menu associations for direct menu access",
		Dependencies: []string{"roles", "menus"},
		Run: func(tx *gorm.DB) error {
			// Get admin role ID
			var adminRole struct{ ID uint }
			if err := tx.Table("roles").Where("code = ?", "admin").First(&adminRole).Error; err != nil {
				return err
			}

			// Get manager role ID (create if not exists, but don't fail if not found)
			var managerRole struct{ ID uint }
			tx.Table("roles").Where("code = ?", "manager").First(&managerRole)

			// Get user role ID (create if not exists, but don't fail if not found)
			var userRole struct{ ID uint }
			tx.Table("roles").Where("code = ?", "user").First(&userRole)

			// Admin role gets access to ALL menus - use direct menu IDs
			// This ensures admin definitely gets all system management submenus
			adminRoleMenus := []map[string]interface{}{
				{"role_id": adminRole.ID, "menu_id": 1},  // Dashboard
				{"role_id": adminRole.ID, "menu_id": 2},  // System (parent)
				{"role_id": adminRole.ID, "menu_id": 3},  // User Management
				{"role_id": adminRole.ID, "menu_id": 4},  // Role Management
				{"role_id": adminRole.ID, "menu_id": 6},  // Menu Management
				{"role_id": adminRole.ID, "menu_id": 7},  // Log (parent)
				{"role_id": adminRole.ID, "menu_id": 8},  // Login Log
				{"role_id": adminRole.ID, "menu_id": 9},  // Operation Log
				{"role_id": adminRole.ID, "menu_id": 10}, // Profile
			}

			// Insert admin role-menu associations
			if err := tx.Table("role_menus").Create(adminRoleMenus).Error; err != nil {
				return err
			}

			// Manager role gets access to specific menus (only if manager role exists)
			if managerRole.ID > 0 {
				managerRoleMenus := []map[string]interface{}{
					{"role_id": managerRole.ID, "menu_id": 1}, // Dashboard
					{"role_id": managerRole.ID, "menu_id": 2}, // System (parent)
					{"role_id": managerRole.ID, "menu_id": 3}, // User Management
					{"role_id": managerRole.ID, "menu_id": 4}, // Role Management
					{"role_id": managerRole.ID, "menu_id": 7}, // Log (parent)
					{"role_id": managerRole.ID, "menu_id": 8}, // Login Log
					{"role_id": managerRole.ID, "menu_id": 9}, // Operation Log
				}
				if err := tx.Table("role_menus").Create(managerRoleMenus).Error; err != nil {
					return err
				}
			}

			// User role gets access to basic menus (only if user role exists)
			if userRole.ID > 0 {
				userRoleMenus := []map[string]interface{}{
					{"role_id": userRole.ID, "menu_id": 1},  // Dashboard
					{"role_id": userRole.ID, "menu_id": 10}, // Profile
				}
				if err := tx.Table("role_menus").Create(userRoleMenus).Error; err != nil {
					return err
				}
			}

			return nil
		},
	})
}
