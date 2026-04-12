package migrations

import (
	"gorm.io/gorm"
)

func init() {
	up := func(tx *gorm.DB) error {
		type UserRole struct {
			ID     uint `gorm:"primarykey"`
			UserID uint `gorm:"not null"`
			RoleID uint `gorm:"not null"`
		}

		if err := tx.AutoMigrate(&UserRole{}); err != nil {
			return err
		}

		for _, pair := range []struct {
			name   string
			create string
		}{
			{"idx_user_roles_user_id", "CREATE INDEX idx_user_roles_user_id ON user_roles(user_id)"},
			{"idx_user_roles_role_id", "CREATE INDEX idx_user_roles_role_id ON user_roles(role_id)"},
		} {
			ok, err := indexExists(tx, "user_roles", pair.name)
			if err != nil {
				return err
			}
			if !ok {
				if err := tx.Exec(pair.create).Error; err != nil {
					return err
				}
			}
		}

		for _, fk := range []struct {
			name string
			sql  string
		}{
			{"fk_user_roles_user_id", "ALTER TABLE user_roles ADD CONSTRAINT fk_user_roles_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"},
			{"fk_user_roles_role_id", "ALTER TABLE user_roles ADD CONSTRAINT fk_user_roles_role_id FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE"},
		} {
			ok, err := fkConstraintExists(tx, "user_roles", fk.name)
			if err != nil {
				return err
			}
			if !ok {
				if err := tx.Exec(fk.sql).Error; err != nil {
					return err
				}
			}
		}

		return nil
	}

	down := func(tx *gorm.DB) error {
		dropForeignKeyBestEffort(tx, "user_roles", "fk_user_roles_user_id")
		dropForeignKeyBestEffort(tx, "user_roles", "fk_user_roles_role_id")
		dropIndexBestEffort(tx, "user_roles", "idx_user_roles_user_id")
		dropIndexBestEffort(tx, "user_roles", "idx_user_roles_role_id")
		return tx.Migrator().DropTable("user_roles")
	}

	Register("create_user_roles_table", NewMigration("20240301000004_create_user_roles_table.go", up, down))
}
