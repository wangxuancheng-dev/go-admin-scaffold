package migrations

import (
	"time"

	"gorm.io/gorm"
)

func init() {
	up := func(tx *gorm.DB) error {
		type Role struct {
			ID          uint           `gorm:"primarykey"`
			Name        string         `gorm:"size:50;not null"`
			Code        string         `gorm:"size:50;not null;unique"`
			Description string         `gorm:"size:255"`
			Status      int            `gorm:"default:1;comment:'Status: 0-inactive, 1-active'"`
			PermList    []string       `gorm:"type:json"`
			CreatedAt   time.Time      `gorm:"type:timestamp"`
			UpdatedAt   time.Time      `gorm:"type:timestamp"`
			DeletedAt   gorm.DeletedAt `gorm:"index;type:timestamp"`
		}

		// Create roles table
		if err := tx.AutoMigrate(&Role{}); err != nil {
			return err
		}

		for _, pair := range []struct {
			name   string
			create string
		}{
			{"idx_roles_code", "CREATE INDEX idx_roles_code ON roles(code)"},
			{"idx_roles_status", "CREATE INDEX idx_roles_status ON roles(status)"},
		} {
			ok, err := indexExists(tx, "roles", pair.name)
			if err != nil {
				return err
			}
			if !ok {
				if err := tx.Exec(pair.create).Error; err != nil {
					return err
				}
			}
		}

		return nil
	}

	down := func(tx *gorm.DB) error {
		dropIndexBestEffort(tx, "roles", "idx_roles_code")
		dropIndexBestEffort(tx, "roles", "idx_roles_status")
		return tx.Migrator().DropTable("roles")
	}

	Register("create_roles_table", NewMigration("20240302_create_roles_table.go", up, down))
}
