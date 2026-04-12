package migrations

import (
	"gorm.io/gorm"
)

func init() {
	up := func(tx *gorm.DB) error {
		type RoleMenu struct {
			RoleID uint `gorm:"primaryKey;column:role_id"`
			MenuID uint `gorm:"primaryKey;column:menu_id"`
		}

		if err := tx.AutoMigrate(&RoleMenu{}); err != nil {
			return err
		}

		for _, pair := range []struct {
			name   string
			create string
		}{
			{"idx_role_menus_role_id", "CREATE INDEX idx_role_menus_role_id ON role_menus(role_id)"},
			{"idx_role_menus_menu_id", "CREATE INDEX idx_role_menus_menu_id ON role_menus(menu_id)"},
			{"idx_role_menus_unique", "CREATE UNIQUE INDEX idx_role_menus_unique ON role_menus(role_id, menu_id)"},
		} {
			ok, err := indexExists(tx, "role_menus", pair.name)
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
			{"fk_role_menus_role_id", "ALTER TABLE role_menus ADD CONSTRAINT fk_role_menus_role_id FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE"},
			{"fk_role_menus_menu_id", "ALTER TABLE role_menus ADD CONSTRAINT fk_role_menus_menu_id FOREIGN KEY (menu_id) REFERENCES menus(id) ON DELETE CASCADE"},
		} {
			ok, err := fkConstraintExists(tx, "role_menus", fk.name)
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
		dropForeignKeyBestEffort(tx, "role_menus", "fk_role_menus_role_id")
		dropForeignKeyBestEffort(tx, "role_menus", "fk_role_menus_menu_id")
		for _, idx := range []string{"idx_role_menus_role_id", "idx_role_menus_menu_id", "idx_role_menus_unique"} {
			dropIndexBestEffort(tx, "role_menus", idx)
		}
		return tx.Migrator().DropTable("role_menus")
	}

	Register("create_role_menus_table", NewMigration("20240311000001_create_role_menus_table.go", up, down))
}
