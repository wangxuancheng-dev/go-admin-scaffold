package database

import (
	"context"
	"fmt"
	"sync"

	"gorm.io/gorm"
)

type ctxKey struct{}

var (
	db   *gorm.DB
	once sync.Once
)

// Init registers the process-level DB handle (used by CLI after SetupDatabase).
func Init(dbConn *gorm.DB) {
	once.Do(func() {
		db = dbConn
	})
}

// GetDB returns the process-level DB, or nil if Init was not called.
// Prefer WithContext / FromContext for new code.
func GetDB() *gorm.DB {
	return db
}

// WithContext attaches db to ctx for command handlers.
func WithContext(ctx context.Context, gdb *gorm.DB) context.Context {
	return context.WithValue(ctx, ctxKey{}, gdb)
}

// FromContext returns the DB from ctx, falling back to GetDB().
func FromContext(ctx context.Context) (*gorm.DB, error) {
	if ctx != nil {
		if v, ok := ctx.Value(ctxKey{}).(*gorm.DB); ok && v != nil {
			return v, nil
		}
	}
	if db != nil {
		return db, nil
	}
	return nil, fmt.Errorf("database is not initialized")
}

// MustFromContext is like FromContext but panics — avoid in libraries; kept for tests.
func MustFromContext(ctx context.Context) *gorm.DB {
	gdb, err := FromContext(ctx)
	if err != nil {
		panic(err)
	}
	return gdb
}

// Close closes the database connection opened via Init.
func Close() error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
