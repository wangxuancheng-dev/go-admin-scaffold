package migrations

import (
	"time"

	"gorm.io/gorm"
)

type CreateUsersTable struct{}

func (m *CreateUsersTable) Up(tx *gorm.DB) error {
	type User struct {
		ID          uint           `gorm:"primarykey"`
		Username    string         `gorm:"size:50;not null;unique;comment:'用户名'"`
		Password    string         `gorm:"size:255;not null;comment:'密码'"`
		Email       string         `gorm:"size:100;not null;unique;comment:'邮箱'"`
		Nickname    string         `gorm:"size:50;comment:'昵称'"`
		Avatar      string         `gorm:"size:255;comment:'头像'"`
		Status      int            `gorm:"default:1;comment:'状态：0-禁用，1-启用'"`
		LastLoginAt *time.Time     `gorm:"type:timestamp;comment:'最后登录时间'"`
		LastLoginIP string         `gorm:"size:50;comment:'最后登录IP'"`
		CreatedAt   time.Time      `gorm:"type:timestamp"`
		UpdatedAt   time.Time      `gorm:"type:timestamp"`
		DeletedAt   gorm.DeletedAt `gorm:"index;type:timestamp"`
	}

	// Create users table
	if err := tx.AutoMigrate(&User{}); err != nil {
		return err
	}

	for _, pair := range []struct {
		name   string
		create string
	}{
		{"idx_users_email", "CREATE INDEX idx_users_email ON users(email)"},
		{"idx_users_status", "CREATE INDEX idx_users_status ON users(status)"},
		{"idx_users_last_login_at", "CREATE INDEX idx_users_last_login_at ON users(last_login_at)"},
	} {
		ok, err := indexExists(tx, "users", pair.name)
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

func (m *CreateUsersTable) Down(tx *gorm.DB) error {
	dropIndexBestEffort(tx, "users", "idx_users_email")
	dropIndexBestEffort(tx, "users", "idx_users_status")
	dropIndexBestEffort(tx, "users", "idx_users_last_login_at")
	return tx.Migrator().DropTable("users")
}

func (m *CreateUsersTable) File() string {
	return "20240301_create_users_table.go"
}

func init() {
	Register("create_users_table", &CreateUsersTable{})
}
