package bootstrap

import (
	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/pkg/cache"
	"go-admin-scaffold/pkg/redis"

	goredis "github.com/redis/go-redis/v9"
)

// SetupRedis initializes the Redis connection and returns the client for the composition root.
func SetupRedis(cfg *config.Config) (*goredis.Client, error) {
	return redis.Setup(&redis.Config{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}

// SetupCache initializes the cache system with the process Redis client and returns it for DI.
func SetupCache(cfg *config.Config, rdb *goredis.Client) (cache.Cache, error) {
	return cache.Setup(&cache.Config{
		Driver:  cfg.Cache.Driver,
		Prefix:  cfg.Cache.Prefix,
		Options: cfg.Cache.Options,
	}, rdb)
}
