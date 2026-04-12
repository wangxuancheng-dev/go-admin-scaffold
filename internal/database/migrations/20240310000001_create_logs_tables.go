package migrations

import (
	"time"

	"gorm.io/gorm"
)

func init() {
	up := func(tx *gorm.DB) error {
		type LoginLog struct {
			ID        uint           `gorm:"primarykey"`
			UserID    uint           `gorm:"index;comment:'用户ID'"`
			Username  string         `gorm:"size:50;comment:'用户名'"`
			IP        string         `gorm:"size:50;comment:'登录IP'"`
			UserAgent string         `gorm:"size:255;comment:'用户代理'"`
			Status    int            `gorm:"default:1;comment:'状态：0-失败，1-成功'"`
			Message   string         `gorm:"size:255;comment:'消息'"`
			LoginTime time.Time      `gorm:"type:timestamp;not null;comment:'登录时间'"`
			CreatedAt time.Time      `gorm:"type:timestamp"`
			UpdatedAt time.Time      `gorm:"type:timestamp"`
			DeletedAt gorm.DeletedAt `gorm:"index;type:timestamp"`
		}

		type OperationLog struct {
			ID            uint           `gorm:"primarykey"`
			UserID        uint           `gorm:"index;comment:'用户ID'"`
			Username      string         `gorm:"size:50;comment:'用户名'"`
			IP            string         `gorm:"size:50;comment:'操作IP'"`
			Method        string         `gorm:"size:20;comment:'请求方法'"`
			Path          string         `gorm:"size:255;comment:'请求路径'"`
			Action        string         `gorm:"size:100;comment:'操作类型'"`
			Module        string         `gorm:"size:100;comment:'模块名称'"`
			BusinessID    string         `gorm:"size:100;comment:'业务ID'"`
			BusinessType  string         `gorm:"size:100;comment:'业务类型'"`
			RequestParams string         `gorm:"type:text;comment:'请求参数'"`
			Status        int            `gorm:"default:1;comment:'状态：0-失败，1-成功'"`
			ErrorMessage  string         `gorm:"size:255;comment:'错误信息'"`
			Duration      int64          `gorm:"comment:'执行时长(毫秒)'"`
			OperationTime time.Time      `gorm:"type:timestamp;not null;comment:'操作时间'"`
			UserAgent     string         `gorm:"size:255;comment:'用户代理'"`
			ReqBody       string         `gorm:"type:text;comment:'请求体'"`
			RespBody      string         `gorm:"type:text;comment:'响应体'"`
			CreatedAt     time.Time      `gorm:"type:timestamp"`
			UpdatedAt     time.Time      `gorm:"type:timestamp"`
			DeletedAt     gorm.DeletedAt `gorm:"index;type:timestamp"`
		}

		if err := tx.Set("gorm:table_options", "").Table("login_logs").AutoMigrate(&LoginLog{}); err != nil {
			return err
		}
		if err := tx.Set("gorm:table_options", "").Table("operation_logs").AutoMigrate(&OperationLog{}); err != nil {
			return err
		}

		for _, pair := range []struct {
			table  string
			name   string
			create string
		}{
			{"login_logs", "idx_login_logs_login_time", "CREATE INDEX idx_login_logs_login_time ON login_logs(login_time)"},
			{"login_logs", "idx_login_logs_status", "CREATE INDEX idx_login_logs_status ON login_logs(status)"},
			{"login_logs", "idx_login_logs_user_id", "CREATE INDEX idx_login_logs_user_id ON login_logs(user_id)"},
		} {
			ok, err := indexExists(tx, pair.table, pair.name)
			if err != nil {
				return err
			}
			if !ok {
				if err := tx.Exec(pair.create).Error; err != nil {
					return err
				}
			}
		}

		indexQueries := []struct {
			name  string
			query string
		}{
			{"idx_operation_logs_operation_time", "CREATE INDEX idx_operation_logs_operation_time ON operation_logs(operation_time)"},
			{"idx_operation_logs_module", "CREATE INDEX idx_operation_logs_module ON operation_logs(module)"},
			{"idx_operation_logs_action", "CREATE INDEX idx_operation_logs_action ON operation_logs(action)"},
			{"idx_operation_logs_status", "CREATE INDEX idx_operation_logs_status ON operation_logs(status)"},
			{"idx_operation_logs_user_id", "CREATE INDEX idx_operation_logs_user_id ON operation_logs(user_id)"},
			{"idx_operation_logs_business_type", "CREATE INDEX idx_operation_logs_business_type ON operation_logs(business_type)"},
		}

		for _, idx := range indexQueries {
			ok, err := indexExists(tx, "operation_logs", idx.name)
			if err != nil {
				return err
			}
			if !ok {
				if err := tx.Exec(idx.query).Error; err != nil {
					return err
				}
			}
		}

		return nil
	}

	down := func(tx *gorm.DB) error {
		for _, pair := range []struct {
			table string
			index string
		}{
			{"login_logs", "idx_login_logs_login_time"},
			{"login_logs", "idx_login_logs_status"},
			{"login_logs", "idx_login_logs_user_id"},
			{"operation_logs", "idx_operation_logs_operation_time"},
			{"operation_logs", "idx_operation_logs_module"},
			{"operation_logs", "idx_operation_logs_action"},
			{"operation_logs", "idx_operation_logs_status"},
			{"operation_logs", "idx_operation_logs_user_id"},
			{"operation_logs", "idx_operation_logs_business_type"},
		} {
			dropIndexBestEffort(tx, pair.table, pair.index)
		}

		if err := tx.Migrator().DropTable("login_logs"); err != nil {
			return err
		}
		return tx.Migrator().DropTable("operation_logs")
	}

	Register("create_logs_tables", NewMigration("20240310000001_create_logs_tables.go", up, down))
}
