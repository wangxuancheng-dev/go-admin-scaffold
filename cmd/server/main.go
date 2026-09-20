package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-admin-scaffold/cmd/server/setup"
	_ "go-admin-scaffold/docs" // swagger docs
	"go-admin-scaffold/internal/commands"
	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/schedule"
	"go-admin-scaffold/pkg/console"
	"go-admin-scaffold/pkg/database"
	"go-admin-scaffold/pkg/locker"
	"go-admin-scaffold/pkg/logger"
)

// @title Go Admin Scaffold API
// @version 1.0
// @description A modern Go admin scaffold API server.
// @host localhost:8080
// @BasePath /api
func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	app, err := setup.InitializeApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	var kernel *schedule.Kernel
	if cfg.SchedulerRunInServer() {
		rdb := app.Container().Redis
		if rdb == nil {
			log.Fatal("redis client is nil")
		}
		redisLocker := locker.NewRedisLocker(rdb)
		manager := console.NewManager()
		manager.Register(commands.NewMigrateCommand())
		manager.Register(commands.NewSeedCommand())
		manager.Register(commands.NewMakeCommand())

		scheduler := schedule.NewScheduler(manager, redisLocker)
		kernel = schedule.NewKernel(scheduler)

		ctx, cancel := context.WithCancel(database.WithContext(context.Background(), app.Container().DB))
		defer cancel()

		go func() {
			if err := kernel.Start(ctx); err != nil {
				log.Printf("Scheduler error: %v", err)
			}
		}()
	} else {
		log.Println("scheduler.run_in_server=false; cron not started (use cmd/scheduler)")
	}

	srv := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: app.Engine(),
	}

	go func() {
		log.Printf("Server is running on %s", cfg.Server.Address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	if kernel != nil {
		log.Println("Shutting down scheduler...")
		kernel.Stop()
	}

	if c := app.Container(); c != nil && c.Redis != nil {
		if err := c.Redis.Close(); err != nil {
			log.Printf("Error closing Redis client: %v", err)
		}
	}

	if err := database.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	if err := logger.Close(); err != nil {
		log.Printf("Error closing logger: %v", err)
	}

	log.Println("Server exited")
}
