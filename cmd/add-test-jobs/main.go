package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"app/internal/config"
	"app/internal/core/jobs"
	"app/pkg/database"
	"app/pkg/queue"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.OpenGorm(database.GormOpenConfig{
		Driver:   cfg.Database.Driver,
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		Username: cfg.Database.Username,
		Password: cfg.Database.Password,
		Database: cfg.Database.Database,
		Charset:  cfg.Database.Charset,
		SSLMode:  cfg.Database.SSLMode,
		TimeZone: cfg.Database.TimeZone,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 创建Redis队列管理器
	redisConfig := queue.Config{
		Driver: "redis",
		Options: map[string]interface{}{
			"connection": fmt.Sprintf("redis://%s:%s/%d", cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB),
			"queue":      "default",
		},
	}

	redisManager, err := queue.NewManager(redisConfig)
	if err != nil {
		log.Fatalf("Failed to create Redis queue manager: %v", err)
	}
	defer redisManager.Close()

	// 创建Database队列管理器
	dbConfig := queue.Config{
		Driver: "database",
		Options: map[string]interface{}{
			"db":    db,
			"queue": "high",
		},
	}

	dbManager, err := queue.NewManager(dbConfig)
	if err != nil {
		log.Fatalf("Failed to create Database queue manager: %v", err)
	}
	defer dbManager.Close()

	ctx := context.Background()

	// 添加一些任务到不同的队列
	fmt.Println("正在添加测试任务...")

	// 添加到 default 队列 (Redis)
	for i := 1; i <= 5; i++ {
		job := &jobs.ExampleJob{
			BaseJob: queue.BaseJob{
				Queue:       "default",
				Attempts:    0,
				MaxAttempts: 3,
				Delay:       0,
				Timeout:     60 * time.Second,
				RetryAfter:  60 * time.Second,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			Message: fmt.Sprintf("Redis default task #%d", i),
		}

		if err := redisManager.Push(ctx, job); err != nil {
			log.Printf("Failed to push job to default queue: %v", err)
		}
	}

	// 添加到 high 队列 (Database)
	for i := 1; i <= 3; i++ {
		job := &jobs.SendWelcomeEmailJob{
			BaseJob: queue.BaseJob{
				Queue:       "high",
				Attempts:    0,
				MaxAttempts: 5,
				Delay:       0,
				Timeout:     30 * time.Second,
				RetryAfter:  30 * time.Second,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			Email:   fmt.Sprintf("user%d@example.com", i),
			Name:    fmt.Sprintf("User %d", i),
			Subject: "Welcome to our platform!",
		}

		if err := dbManager.Push(ctx, job); err != nil {
			log.Printf("Failed to push job to high queue: %v", err)
		}
	}

	// 添加一些延迟任务到 low 队列
	redisConfig.Options["queue"] = "low"
	lowManager, err := queue.NewManager(redisConfig)
	if err != nil {
		log.Fatalf("Failed to create low queue manager: %v", err)
	}
	defer lowManager.Close()

	for i := 1; i <= 2; i++ {
		job := &jobs.CleanupJob{
			BaseJob: queue.BaseJob{
				Queue:       "low",
				Attempts:    0,
				MaxAttempts: 2,
				Delay:       time.Duration(i*10) * time.Second,
				Timeout:     120 * time.Second,
				RetryAfter:  120 * time.Second,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			Target:    fmt.Sprintf("/tmp/cleanup-%d", i),
			StartTime: time.Now(),
			EndTime:   time.Now().Add(24 * time.Hour),
		}

		if err := lowManager.Later(ctx, job, time.Duration(i*10)*time.Second); err != nil {
			log.Printf("Failed to push delayed job to low queue: %v", err)
		}
	}

	fmt.Println("✅ 测试任务添加完成!")
	fmt.Println("- default 队列 (Redis): 5 个任务")
	fmt.Println("- high 队列 (Database): 3 个任务")
	fmt.Println("- low 队列 (Redis): 2 个延迟任务")
}
