package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"app/internal/config"
	"app/internal/core/jobs"
	"app/internal/core/services"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	queueService, err := services.NewQueueService(cfg)
	if err != nil {
		log.Fatalf("Failed to create queue service: %v", err)
	}

	// 启动队列服务
	if err := queueService.Start(); err != nil {
		log.Fatalf("Failed to start queue service: %v", err)
	}
	defer queueService.Stop()

	// 创建上下文
	ctx := context.Background()

	// 推送示例任务
	exampleJob := jobs.NewExampleJob("Hello, Queue!")
	if err := queueService.Push(ctx, exampleJob); err != nil {
		log.Printf("Failed to push example job: %v", err)
	}

	// 推送欢迎邮件任务
	welcomeJob := jobs.NewSendWelcomeEmailJob("user@example.com", "John Doe")
	if err := queueService.Push(ctx, welcomeJob); err != nil {
		log.Printf("Failed to push welcome email job: %v", err)
	}

	// 推送文件处理任务
	uploadJob := jobs.NewProcessUploadJob("file123", "example.pdf", 1024*1024)
	if err := queueService.Push(ctx, uploadJob); err != nil {
		log.Printf("Failed to push upload job: %v", err)
	}

	// 推送延迟清理任务
	options, _ := json.Marshal(map[string]any{
		"type":    "logs",
		"pattern": "*.log",
	})
	cleanupJob := jobs.NewCleanupJob("logs", options, time.Now().AddDate(0, -1, 0), time.Now())
	if err := queueService.Later(ctx, cleanupJob, 5*time.Minute); err != nil {
		log.Printf("Failed to push cleanup job: %v", err)
	}

	// 获取队列大小
	size, err := queueService.Size(ctx, "default")
	if err != nil {
		log.Printf("Failed to get queue size: %v", err)
	} else {
		log.Printf("Default queue size: %d", size)
	}

	// 等待一段时间，让任务处理完成
	time.Sleep(10 * time.Second)

	// 获取活动队列列表
	queues := queueService.GetActiveQueues()
	log.Printf("Active queues: %v", queues)

	// 获取工作进程数量
	workerCount := queueService.GetWorkerCount()
	log.Printf("Worker count: %d", workerCount)
}
