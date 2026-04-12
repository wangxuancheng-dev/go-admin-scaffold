package migrations

import (
	"time"

	"gorm.io/gorm"
)

func init() {
	up := func(tx *gorm.DB) error {
		type Todo struct {
			ID        uint           `gorm:"primarykey"`
			Title     string         `gorm:"size:255;not null;comment:'标题'"`
			Content   string         `gorm:"type:text;comment:'内容'"`
			Status    int            `gorm:"default:0;comment:'状态：0-未完成，1-已完成'"`
			Priority  int            `gorm:"default:0;comment:'优先级：0-低，1-中，2-高'"`
			DueDate   *time.Time     `gorm:"type:timestamp;comment:'截止日期'"`
			UserID    uint           `gorm:"not null;comment:'创建者ID'"`
			CreatedAt time.Time      `gorm:"type:timestamp"`
			UpdatedAt time.Time      `gorm:"type:timestamp"`
			DeletedAt gorm.DeletedAt `gorm:"index;type:timestamp"`
		}

		if err := tx.AutoMigrate(&Todo{}); err != nil {
			return err
		}

		for _, pair := range []struct {
			name   string
			create string
		}{
			{"idx_todos_user_id", "CREATE INDEX idx_todos_user_id ON todos(user_id)"},
			{"idx_todos_status", "CREATE INDEX idx_todos_status ON todos(status)"},
			{"idx_todos_due_date", "CREATE INDEX idx_todos_due_date ON todos(due_date)"},
		} {
			ok, err := indexExists(tx, "todos", pair.name)
			if err != nil {
				return err
			}
			if !ok {
				if err := tx.Exec(pair.create).Error; err != nil {
					return err
				}
			}
		}

		ok, err := fkConstraintExists(tx, "todos", "fk_todos_user_id")
		if err != nil {
			return err
		}
		if !ok {
			if err := tx.Exec("ALTER TABLE todos ADD CONSTRAINT fk_todos_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE").Error; err != nil {
				return err
			}
		}

		return nil
	}

	down := func(tx *gorm.DB) error {
		dropForeignKeyBestEffort(tx, "todos", "fk_todos_user_id")
		dropIndexBestEffort(tx, "todos", "idx_todos_user_id")
		dropIndexBestEffort(tx, "todos", "idx_todos_status")
		dropIndexBestEffort(tx, "todos", "idx_todos_due_date")
		return tx.Migrator().DropTable("todos")
	}

	Register("create_todos_table", NewMigration("2025_06_01_185848_create_todos_table.go", up, down))
}
