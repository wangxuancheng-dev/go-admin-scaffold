package main

import (
	"context"
	"fmt"
	"log"

	"go-admin-scaffold/internal/bootstrap"
	"go-admin-scaffold/internal/commands"
	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/schedule"
	"go-admin-scaffold/pkg/console"
	"go-admin-scaffold/pkg/database"
	"go-admin-scaffold/pkg/locker"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	db, err := bootstrap.SetupDatabase(cfg)
	if err != nil {
		log.Fatal(err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	redisLocker := locker.NewRedisLocker(redisClient)
	manager := console.NewManager()
	manager.Register(commands.NewMakeCommand())
	manager.Register(commands.NewMigrateCommand())
	manager.Register(commands.NewSeedCommand())

	scheduler := schedule.NewScheduler(manager, redisLocker)
	kernel := schedule.NewKernel(scheduler)
	manager.Register(commands.NewScheduleRunCommand(kernel))

	ctx := database.WithContext(context.Background(), db)
	if err := manager.RunFromArgsWithContext(ctx); err != nil {
		log.Fatal(err)
	}
}
