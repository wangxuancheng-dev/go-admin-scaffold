package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// CustomTime 是一个自定义时间类型，用于统一时间格式
type CustomTime time.Time

// MarshalJSON 实现 json.Marshaler 接口
func (t CustomTime) MarshalJSON() ([]byte, error) {
	tm := time.Time(t)
	if tm.IsZero() {
		return []byte("null"), nil
	}
	// 使用本地时区格式化时间
	return json.Marshal(tm.Local().Format("2006-01-02 15:04:05"))
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (t *CustomTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	// 尝试多种时间格式解析
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05+08:00",
		time.RFC3339,
	}

	for _, format := range formats {
		if tm, err := time.Parse(format, s); err == nil {
			*t = CustomTime(tm)
			return nil
		}
	}
	return fmt.Errorf("invalid time format: %s", s)
}

// Value 实现 driver.Valuer 接口
func (t CustomTime) Value() (driver.Value, error) {
	return time.Time(t), nil
}

// Scan 实现 sql.Scanner 接口
func (t *CustomTime) Scan(value interface{}) error {
	if value == nil {
		*t = CustomTime(time.Time{})
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		*t = CustomTime(v)
		return nil
	case string:
		tm, err := time.Parse("2006-01-02 15:04:05", v)
		if err != nil {
			return err
		}
		*t = CustomTime(tm)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into CustomTime", value)
	}
}

// String 返回格式化的时间字符串
func (t CustomTime) String() string {
	return time.Time(t).Local().Format("2006-01-02 15:04:05")
}

// BaseModel is the standard embedded model for soft delete.
//
// Convention: embed BaseModel on GORM models that should use soft delete (default queries exclude rows with DeletedAt set).
// To include deleted rows in a query, use db.Unscoped(). To permanently remove a row, use Unscoped().Delete(...).
type BaseModel struct {
	ID uint `gorm:"primarykey" json:"id"`
	// CreatedAt CustomTime `json:"created_at"`
	CreatedAt time.Time `json:"created_at" gorm:"type:timestamp"`
	// UpdatedAt CustomTime `json:"updated_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:timestamp"`
	// DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;type:timestamp"`
}

// Pagination represents common pagination parameters
type Pagination struct {
	Page     int   `json:"page" form:"page"`
	PageSize int   `json:"page_size" form:"page_size"`
	Total    int64 `json:"total"`
}

func (p *Pagination) GetOffset() int {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 10
	}
	return (p.Page - 1) * p.PageSize
}

func (p *Pagination) GetLimit() int {
	if p.PageSize < 1 {
		p.PageSize = 10
	}
	return p.PageSize
}
