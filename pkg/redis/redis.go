package redis

import (
	"context"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	client *redis.Client
	once   sync.Once
)

// Config represents Redis configuration.
type Config struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// Setup initializes Redis once per process and returns the client for injection.
func Setup(cfg *Config) (*redis.Client, error) {
	var setupErr error
	once.Do(func() {
		if cfg == nil {
			setupErr = fmt.Errorf("redis config is required")
			return
		}
		c := redis.NewClient(&redis.Options{
			Addr:     cfg.Host + ":" + cfg.Port,
			Password: cfg.Password,
			DB:       cfg.DB,
		})
		if err := c.Ping(context.Background()).Err(); err != nil {
			_ = c.Close()
			setupErr = err
			return
		}
		client = c
	})
	if setupErr != nil {
		return nil, setupErr
	}
	if client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}
	return client, nil
}

// Close closes the Redis client opened via Setup.
func Close() error {
	if client != nil {
		return client.Close()
	}
	return nil
}
