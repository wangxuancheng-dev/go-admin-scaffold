package database

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormOpenConfig holds everything needed to open a *gorm.DB without importing internal/config.
type GormOpenConfig struct {
	Driver   string
	Host     string
	Port     string
	Username string
	Password string
	Database string
	Charset  string
	SSLMode  string // postgres: disable, require, verify-full, etc.
	TimeZone string // postgres DSN TimeZone param, e.g. UTC, Asia/Shanghai
}

// OpenGorm opens a database connection for mysql or postgres.
func OpenGorm(cfg GormOpenConfig) (*gorm.DB, error) {
	driver := strings.ToLower(strings.TrimSpace(cfg.Driver))
	if driver == "" {
		driver = "mysql"
	}

	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	switch driver {
	case "mysql":
		return gorm.Open(mysql.Open(mysqlDSN(cfg)), gormCfg)
	case "postgres", "postgresql", "pg":
		return gorm.Open(postgres.Open(postgresDSN(cfg)), gormCfg)
	default:
		return nil, fmt.Errorf("unsupported database driver %q (use mysql or postgres)", cfg.Driver)
	}
}

func mysqlDSN(cfg GormOpenConfig) string {
	charset := strings.TrimSpace(cfg.Charset)
	if charset == "" {
		charset = "utf8mb4"
	}
	port := strings.TrimSpace(cfg.Port)
	if port == "" {
		port = "3306"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		port,
		cfg.Database,
		charset,
	)
}

func postgresDSN(cfg GormOpenConfig) string {
	port := strings.TrimSpace(cfg.Port)
	if port == "" {
		port = "5432"
	}
	ssl := strings.TrimSpace(cfg.SSLMode)
	if ssl == "" {
		ssl = "disable"
	}
	tz := strings.TrimSpace(cfg.TimeZone)
	if tz == "" {
		tz = "UTC"
	}

	host := cfg.Host
	if host == "" {
		host = "localhost"
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.Username, cfg.Password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + cfg.Database,
	}
	q := u.Query()
	q.Set("sslmode", ssl)
	q.Set("TimeZone", tz)
	if ch := strings.TrimSpace(cfg.Charset); ch != "" {
		q.Set("client_encoding", ch)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
