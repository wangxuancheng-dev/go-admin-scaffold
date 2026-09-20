package bootstrap

import (
	"fmt"
	"time"

	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/pkg/database"

	"gorm.io/gorm"
)

// SetupDatabase initializes the database connection and registers the global handle for CLI tools.
// Prefer using the returned *gorm.DB in the HTTP composition root.
func SetupDatabase(cfg *config.Config) (*gorm.DB, error) {
	db, err := database.OpenGorm(database.GormOpenConfig{
		Driver:   cfg.Database.Driver,
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		Username: cfg.Database.Username,
		Password: cfg.Database.Password,
		Database: cfg.Database.Database,
		Charset:  cfg.Database.Charset,
		SSLMode:  cfg.Database.SSLMode,
		TimeZone: cfg.Database.TimeZone,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)

	database.Init(db)
	return db, nil
}
