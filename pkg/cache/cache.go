package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	ErrKeyNotFound = errors.New("key not found in cache")
	ErrKeyExpired  = errors.New("key has expired")
)

// Cache interface defines the methods that any cache implementation must provide
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// RedisCache implements Cache interface using an injected Redis client.
type RedisCache struct {
	client *goredis.Client
	prefix string
}

// Config represents cache configuration
type Config struct {
	Driver  string                 `mapstructure:"driver"`
	Prefix  string                 `mapstructure:"prefix"`
	Options map[string]interface{} `mapstructure:"options"`
}

var defaultCache Cache

// Setup initializes the cache system with an explicit Redis client when driver is redis.
func Setup(cfg *Config, rdb *goredis.Client) error {
	switch cfg.Driver {
	case "redis":
		if rdb == nil {
			return fmt.Errorf("redis client is required for cache driver redis")
		}
		defaultCache = &RedisCache{client: rdb, prefix: cfg.Prefix}
		return nil
	case "file", "":
		// File cache via Manager.Store; keep Default optional.
		defaultCache = nil
		return nil
	default:
		return fmt.Errorf("unsupported cache driver: %s", cfg.Driver)
	}
}

// Default returns the process-level cache set by Setup.
// Prefer injecting Cache (or *redis.Client) via the composition root for HTTP paths;
// Default is for CLI / legacy helpers that are not wired through Container.
func Default() Cache {
	return defaultCache
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, c.prefix+key).Result()
}

func (c *RedisCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return c.client.Set(ctx, c.prefix+key, value, expiration).Err()
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, c.prefix+key).Err()
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, c.prefix+key).Result()
	return result > 0, err
}

// Store defines the interface for cache implementations
type Store interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	Remember(ctx context.Context, key string, expiration time.Duration, getter func() (interface{}, error)) (interface{}, error)
	Has(ctx context.Context, key string) bool
	Increment(ctx context.Context, key string) error
	Decrement(ctx context.Context, key string) error
	Close() error
}

func (c *Config) GetFilePath() string {
	if path, ok := c.Options["file_path"].(string); ok {
		return path
	}
	return "storage/cache"
}

func (c *Config) GetRedisConfig() RedisConfig {
	config := RedisConfig{Host: "localhost", Port: 6379}
	if host, ok := c.Options["host"].(string); ok {
		config.Host = host
	}
	if port, ok := c.Options["port"].(int); ok {
		config.Port = port
	}
	if password, ok := c.Options["password"].(string); ok {
		config.Password = password
	}
	if db, ok := c.Options["db"].(int); ok {
		config.DB = db
	}
	return config
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

var (
	DefaultExpiration = 24 * time.Hour
	NoExpiration      = time.Duration(0)
)

type Manager struct {
	config *Config
	stores map[string]Store
}

func NewManager(config *Config) *Manager {
	return &Manager{config: config, stores: make(map[string]Store)}
}

func (m *Manager) Store(driver string) (Store, error) {
	if store, exists := m.stores[driver]; exists {
		return store, nil
	}
	var store Store
	var err error
	switch driver {
	case "file":
		store, err = NewFileStore(m.config)
	case "redis":
		store, err = NewRedisStore(m.config)
	default:
		store, err = NewFileStore(m.config)
	}
	if err != nil {
		return nil, err
	}
	m.stores[driver] = store
	return store, nil
}

func (m *Manager) Close() error {
	for _, store := range m.stores {
		if err := store.Close(); err != nil {
			return err
		}
	}
	return nil
}
