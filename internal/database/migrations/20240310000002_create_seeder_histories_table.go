package migrations

import (
	"time"

	"gorm.io/gorm"
)

func init() {
	up := func(tx *gorm.DB) error {
		type SeederHistory struct {
			ID         uint      `gorm:"primarykey"`
			Name       string    `gorm:"size:255;not null;unique"`
			ExecutedAt time.Time `gorm:"type:timestamp;not null"`
		}

		exists, err := tableExists(tx, "seeder_histories")
		if err != nil {
			return err
		}

		if !exists {
			if err := tx.AutoMigrate(&SeederHistory{}); err != nil {
				return err
			}
			if err := tx.Exec("CREATE INDEX idx_seeder_histories_executed_at ON seeder_histories(executed_at)").Error; err != nil {
				return err
			}
			return nil
		}

		if err := tx.AutoMigrate(&SeederHistory{}); err != nil {
			return err
		}
		if isPostgres(tx) {
			_ = tx.Exec(`ALTER TABLE seeder_histories ALTER COLUMN executed_at SET NOT NULL`).Error
		} else {
			_ = tx.Exec("ALTER TABLE seeder_histories MODIFY COLUMN executed_at TIMESTAMP NOT NULL").Error
		}
		return nil
	}

	down := func(tx *gorm.DB) error {
		dropIndexBestEffort(tx, "seeder_histories", "idx_seeder_histories_executed_at")
		return tx.Migrator().DropTable("seeder_histories")
	}

	Register("create_seeder_histories_table", NewMigration("20240310000002_create_seeder_histories_table.go", up, down))
}
