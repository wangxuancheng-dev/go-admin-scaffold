package main

import (
	"fmt"
	"log"

	"app/internal/commands"
	"app/internal/config"
	"app/internal/schedule"
	"app/pkg/console"
	"app/pkg/locker"

	"github.com/redis/go-redis/v9"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	// Create Redis locker
	redisLocker := locker.NewRedisLocker(redisClient)

	// Create command manager
	manager := console.NewManager()

	// Register built-in commands
	manager.Register(commands.NewMakeCommand())
	manager.Register(commands.NewMigrateCommand())
	manager.Register(commands.NewSeedCommand())

	// Create scheduler
	scheduler := schedule.NewScheduler(manager, redisLocker)
	kernel := schedule.NewKernel(scheduler)

	// Register scheduler command
	manager.Register(commands.NewScheduleRunCommand(kernel))

	// Run command from arguments
	if err := manager.RunFromArgs(); err != nil {
		log.Fatal(err)
	}
}
