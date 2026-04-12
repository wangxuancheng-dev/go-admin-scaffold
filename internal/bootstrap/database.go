package bootstrap

import (
	"fmt"
	"time"

	"app/internal/config"
	"app/pkg/database"
)

// SetupDatabase initializes the database connection (MySQL or PostgreSQL per cfg.Database.Driver).
func SetupDatabase(cfg *config.Config) error {
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
		return fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)

	database.Init(db)

	return nil
}
