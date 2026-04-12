package seeders

import (
	"app/internal/database/seeder"
	"encoding/json"

	"gorm.io/gorm"
)

func init() {
	Register("roles", &seeder.Seeder{
		Name:        "roles",
		Description: "Create default roles",
		Run: func(tx *gorm.DB) error {
			roles := []map[string]interface{}{
				{
					"name":        "Administrator",
					"code":        "admin",
					"description": "System administrator with full access",
					"status":      1,
					"perm_list":   json.RawMessage(`["*"]`),
				},
				{
					"name":        "Manager",
					"code":        "manager",
					"description": "Department manager with limited access",
					"status":      1,
					"perm_list":   json.RawMessage(`["dashboard:view", "users:view", "users:edit", "roles:view", "logs:view"]`),
				},
				{
					"name":        "User",
					"code":        "user",
					"description": "Regular user with basic access",
					"status":      1,
					"perm_list":   json.RawMessage(`["dashboard:view", "profile:view", "profile:edit"]`),
				},
			}

			return tx.Table("roles").Create(roles).Error
		},
	})
}
