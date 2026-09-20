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

// Init registers the process-level DB handle for Close() after SetupDatabase.
func Init(dbConn *gorm.DB) {
	once.Do(func() {
		db = dbConn
	})
}

// WithContext attaches db to ctx for command handlers.
func WithContext(ctx context.Context, gdb *gorm.DB) context.Context {
	return context.WithValue(ctx, ctxKey{}, gdb)
}

// FromContext returns the DB attached via WithContext.
func FromContext(ctx context.Context) (*gorm.DB, error) {
	if ctx != nil {
		if v, ok := ctx.Value(ctxKey{}).(*gorm.DB); ok && v != nil {
			return v, nil
		}
	}
	return nil, fmt.Errorf("database is not in context; use database.WithContext")
}

// MustFromContext is like FromContext but panics.
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
