package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"app/cmd/server/setup"
	_ "app/docs" // 导入 swagger 文档
	"app/internal/commands"
	"app/internal/config"
	"app/internal/schedule"
	"app/pkg/console"
	"app/pkg/database"
	"app/pkg/locker"
	"app/pkg/logger"
	"app/pkg/redis"
)

// @title Go Admin Scaffold API
// @version 1.0
// @description A modern Go admin scaffold API server.
// @host localhost:8080
// @BasePath /api
func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the HTTP server (database, Redis, cache, logger, routes)
	app, err := setup.InitializeApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Scheduler uses the same Redis client as the app
	redisLocker := locker.NewRedisLocker(redis.GetClient())

	manager := console.NewManager()
	manager.Register(commands.NewHelloWorldCommand())
	manager.Register(commands.NewMigrateCommand())
	manager.Register(commands.NewSeedCommand())
	manager.Register(commands.NewSendEmailsCommand())
	manager.Register(commands.NewMakeCommand())

	scheduler := schedule.NewScheduler(manager, redisLocker)
	kernel := schedule.NewKernel(scheduler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := kernel.Start(ctx); err != nil {
			log.Printf("Scheduler error: %v", err)
		}
	}()

	// Start HTTP server in a goroutine
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

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Shutdown gracefully
	log.Println("Shutting down server...")

	// Create a deadline for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Stop scheduler
	log.Println("Shutting down scheduler...")
	kernel.Stop()

	if err := redis.Close(); err != nil {
		log.Printf("Error closing Redis client: %v", err)
	}

	if err := database.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	if err := logger.Close(); err != nil {
		log.Printf("Error closing logger: %v", err)
	}

	log.Println("Server exited")
}
