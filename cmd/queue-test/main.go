package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"app/internal/config"
	"app/internal/core/jobs"
	"app/pkg/database"
	"app/pkg/queue"

	"gorm.io/gorm"
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

	// 测试Redis驱动
	fmt.Println("=== 测试 Redis 驱动 ===")
	testRedisDriver(cfg)

	// 测试Database驱动
	fmt.Println("\n=== 测试 Database 驱动 ===")
	testDatabaseDriver(cfg, db)
}

func testRedisDriver(cfg *config.Config) {
	// 创建Redis队列
	queueConfig := queue.Config{
		Driver: "redis",
		Options: map[string]interface{}{
			"connection": fmt.Sprintf("redis://%s:%s/%d", cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB),
			"queue":      "test-redis",
		},
	}

	manager, err := queue.NewManager(queueConfig)
	if err != nil {
		log.Printf("Failed to create Redis queue manager: %v", err)
		return
	}
	defer manager.Close()

	testQueue(manager, "Redis")
}

func testDatabaseDriver(cfg *config.Config, db *gorm.DB) {
	// 创建Database队列
	queueConfig := queue.Config{
		Driver: "database",
		Options: map[string]interface{}{
			"db":    db,
			"queue": "test-database",
		},
	}

	manager, err := queue.NewManager(queueConfig)
	if err != nil {
		log.Printf("Failed to create Database queue manager: %v", err)
		return
	}
	defer manager.Close()

	testQueue(manager, "Database")
}

func testQueue(manager *queue.Manager, driverName string) {
	ctx := context.Background()
	queueName := fmt.Sprintf("test-%s", driverName)

	fmt.Printf("开始测试 %s 驱动...\n", driverName)

	// 清空队列
	err := manager.Clear(ctx, queueName)
	if err != nil {
		log.Printf("Failed to clear queue: %v", err)
		return
	}
	fmt.Printf("✓ 清空队列成功\n")

	// 1. 测试基本任务推送和获取
	fmt.Printf("\n1. 测试基本任务推送和获取:\n")

	// 创建测试任务
	job := &jobs.ExampleJob{
		BaseJob: queue.BaseJob{
			Queue:       queueName,
			Attempts:    0,
			MaxAttempts: 3,
			Delay:       0,
			Timeout:     60 * time.Second,
			RetryAfter:  60 * time.Second,
			Backoff:     []time.Duration{60 * time.Second, 300 * time.Second},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		Message: "Hello from " + driverName + " queue!",
	}

	// 推送任务
	err = manager.Push(ctx, job)
	if err != nil {
		log.Printf("Failed to push job: %v", err)
		return
	}
	fmt.Printf("✓ 推送任务成功\n")

	// 检查队列大小
	size, err := manager.Size(ctx, queueName)
	if err != nil {
		log.Printf("Failed to get queue size: %v", err)
		return
	}
	fmt.Printf("✓ 队列大小: %d\n", size)

	// 获取任务
	retrievedJob, err := manager.Pop(ctx, queueName)
	if err != nil {
		log.Printf("Failed to pop job: %v", err)
		return
	}
	fmt.Printf("✓ 获取任务成功: %s\n", string(retrievedJob.GetPayload()))

	// 删除任务
	err = manager.Delete(ctx, queueName, retrievedJob)
	if err != nil {
		log.Printf("Failed to delete job: %v", err)
		return
	}
	fmt.Printf("✓ 删除任务成功\n")

	// 2. 测试延迟任务
	fmt.Printf("\n2. 测试延迟任务:\n")

	delayedJob := &jobs.ExampleJob{
		BaseJob: queue.BaseJob{
			Queue:       queueName,
			Attempts:    0,
			MaxAttempts: 3,
			Delay:       5 * time.Second, // 延迟5秒
			Timeout:     60 * time.Second,
			RetryAfter:  60 * time.Second,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		Message: "Delayed job from " + driverName,
	}

	// 推送延迟任务
	err = manager.Later(ctx, delayedJob, 3*time.Second)
	if err != nil {
		log.Printf("Failed to push delayed job: %v", err)
		return
	}
	fmt.Printf("✓ 推送延迟任务成功 (3秒后可用)\n")

	// 立即尝试获取任务（应该为空）
	_, err = manager.Pop(ctx, queueName)
	if err == queue.ErrQueueEmpty {
		fmt.Printf("✓ 延迟任务暂不可用 (符合预期)\n")
	} else if err != nil {
		log.Printf("Unexpected error: %v", err)
		return
	} else {
		fmt.Printf("⚠ 延迟任务立即可用 (可能不符合预期)\n")
	}

	// 等待并再次尝试
	fmt.Printf("等待4秒...\n")
	time.Sleep(4 * time.Second)

	delayedRetrievedJob, err := manager.Pop(ctx, queueName)
	if err == queue.ErrQueueEmpty {
		fmt.Printf("⚠ 延迟任务仍不可用\n")
	} else if err != nil {
		log.Printf("Failed to pop delayed job: %v", err)
		return
	} else {
		fmt.Printf("✓ 延迟任务可用: %s\n", string(delayedRetrievedJob.GetPayload()))

		// 清理延迟任务
		err = manager.Delete(ctx, queueName, delayedRetrievedJob)
		if err != nil {
			log.Printf("Failed to delete delayed job: %v", err)
		}
	}

	// 3. 测试任务重试和释放
	fmt.Printf("\n3. 测试任务重试:\n")

	retryJob := &jobs.ExampleJob{
		BaseJob: queue.BaseJob{
			Queue:       queueName,
			Attempts:    1,
			MaxAttempts: 3,
			Delay:       0,
			Timeout:     60 * time.Second,
			RetryAfter:  2 * time.Second,
			Backoff:     []time.Duration{2 * time.Second, 4 * time.Second},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		Message: "Retry job from " + driverName,
	}

	// 推送任务
	err = manager.Push(ctx, retryJob)
	if err != nil {
		log.Printf("Failed to push retry job: %v", err)
		return
	}

	// 获取任务
	retryRetrievedJob, err := manager.Pop(ctx, queueName)
	if err != nil {
		log.Printf("Failed to pop retry job: %v", err)
		return
	}
	fmt.Printf("✓ 获取重试任务成功\n")

	// 释放任务（模拟失败重试）
	err = manager.Release(ctx, queueName, retryRetrievedJob, 1*time.Second)
	if err != nil {
		log.Printf("Failed to release job: %v", err)
		return
	}
	fmt.Printf("✓ 释放任务成功 (1秒后重新可用)\n")

	// 4. 测试原始数据推送
	fmt.Printf("\n4. 测试原始数据推送:\n")

	rawData := map[string]interface{}{
		"type":    "email",
		"to":      "test@example.com",
		"subject": "Test Email",
		"body":    "This is a test email from " + driverName,
	}

	rawPayload, _ := json.Marshal(rawData)

	err = manager.PushRaw(ctx, queueName, rawPayload, map[string]interface{}{
		"delay":        2 * time.Second,
		"max_attempts": 3,
		"timeout":      30 * time.Second,
	})
	if err != nil {
		log.Printf("Failed to push raw job: %v", err)
		return
	}
	fmt.Printf("✓ 推送原始数据成功\n")

	// 5. 最终检查队列大小
	finalSize, err := manager.Size(ctx, queueName)
	if err != nil {
		log.Printf("Failed to get final queue size: %v", err)
		return
	}
	fmt.Printf("\n✓ 最终队列大小: %d\n", finalSize)

	// 清空队列
	err = manager.Clear(ctx, queueName)
	if err != nil {
		log.Printf("Failed to clear queue: %v", err)
		return
	}
	fmt.Printf("✓ 清空队列成功\n")

	fmt.Printf("\n%s 驱动测试完成! ✅\n", driverName)
}
