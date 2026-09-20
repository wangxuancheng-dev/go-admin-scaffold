package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-admin-scaffold/internal/bootstrap"
	"go-admin-scaffold/internal/commands"
	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/schedule"
	"go-admin-scaffold/pkg/console"
	"go-admin-scaffold/pkg/database"
	"go-admin-scaffold/pkg/locker"
	"go-admin-scaffold/pkg/logger"
)

// Standalone cron process. Prefer this in production with scheduler.run_in_server=false.
func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	if err := logger.Setup(&logger.Config{
		Level:      cfg.Log.Level,
		Filename:   cfg.Log.Filename,
		MaxSize:    cfg.Log.MaxSize,
		MaxAge:     cfg.Log.MaxAge,
		MaxBackups: cfg.Log.MaxBackups,
		Compress:   cfg.Log.Compress,
		Daily:      cfg.Log.Daily,
		Timezone:   cfg.Log.Timezone,
	}); err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = logger.Close()
	}()

	db, err := bootstrap.SetupDatabase(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = database.Close()
	}()

	rdb, err := bootstrap.SetupRedis(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = rdb.Close()
	}()

	redisLocker := locker.NewRedisLocker(rdb)
	manager := console.NewManager()
	manager.Register(commands.NewMigrateCommand())
	manager.Register(commands.NewSeedCommand())
	manager.Register(commands.NewMakeCommand())

	scheduler := schedule.NewScheduler(manager, redisLocker)
	kernel := schedule.NewKernel(scheduler)

	ctx, cancel := context.WithCancel(database.WithContext(context.Background(), db))
	defer cancel()

	go func() {
		if err := kernel.Start(ctx); err != nil {
			log.Printf("Scheduler error: %v", err)
		}
	}()

	log.Println("Scheduler process running (ctrl+c to stop)")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down scheduler...")
	cancel()
	kernel.Stop()
	log.Println("Scheduler exited")
}
