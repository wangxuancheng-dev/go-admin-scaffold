package storage

import (
	"fmt"
	"io"
	"strings"
)

// Storage defines the file storage interface used by upload handlers.
type Storage interface {
	// Put stores content at path and returns a publicly accessible URL.
	Put(path string, reader io.Reader) (string, error)
	// Delete removes a file at path.
	Delete(path string) error
	// URL returns the public URL for a stored path.
	URL(path string) string
}

// Config holds storage driver configuration.
type Config struct {
	Driver    string
	LocalPath string
	S3Config  *S3Config
}

// S3Config holds S3-compatible storage settings.
type S3Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
	UseSSL          bool
}

// NewStorage creates a storage backend from config.
func NewStorage(cfg *Config) (Storage, error) {
	if cfg == nil {
		return nil, fmt.Errorf("storage config is required")
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Driver)) {
	case "", "local":
		return NewLocalStorage(cfg.LocalPath)
	case "s3":
		if cfg.S3Config == nil {
			return nil, fmt.Errorf("s3 config is required when driver is s3")
		}
		return NewS3Storage(cfg.S3Config)
	default:
		return nil, fmt.Errorf("unsupported storage driver: %s", cfg.Driver)
	}
}
