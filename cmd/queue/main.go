package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/core/queuesvc"
)

var (
	configFile  string
	queueName   string
	listQueues  bool
	clearQueue  bool
	statusQueue bool
)

func init() {
	flag.StringVar(&configFile, "config", "configs/config.yaml", "配置文件路径")
	flag.StringVar(&queueName, "queue", "", "队列名称")
	flag.BoolVar(&listQueues, "list", false, "列出所有队列")
	flag.BoolVar(&clearQueue, "clear", false, "清空队列")
	flag.BoolVar(&statusQueue, "status", false, "查询队列状态")
}

func main() {
	flag.Parse()

	cfg, err := config.LoadConfigFromFile(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	queueService, err := queuesvc.NewQueueService(cfg)
	if err != nil {
		log.Fatalf("Failed to create queue service: %v", err)
	}

	switch {
	case listQueues:
		queues := queueService.GetActiveQueues()
		if len(queues) == 0 {
			fmt.Println("No active queues")
			return
		}
		fmt.Println("Active queues:")
		for _, name := range queues {
			fmt.Printf("- %s\n", name)
		}

	case clearQueue:
		if queueName == "" {
			log.Fatal("Queue name is required")
		}
		ctx := context.Background()
		if err := queueService.Clear(ctx, queueName); err != nil {
			log.Fatalf("Failed to clear queue %s: %v", queueName, err)
		}
		fmt.Printf("Queue %s cleared\n", queueName)

	case statusQueue:
		ctx := context.Background()
		if queueName == "" {
			queues := queueService.GetActiveQueues()
			if len(queues) == 0 {
				fmt.Println("No active queues")
				return
			}
			fmt.Println("Queue Status:")
			fmt.Println("=============")
			for _, name := range queues {
				size, err := queueService.Size(ctx, name)
				if err != nil {
					fmt.Printf("%-20s: Error - %v\n", name, err)
				} else {
					fmt.Printf("%-20s: %d jobs\n", name, size)
				}
			}
			return
		}
		size, err := queueService.Size(ctx, queueName)
		if err != nil {
			log.Fatalf("Failed to get queue size for %s: %v", queueName, err)
		}
		fmt.Printf("Queue: %s\n", queueName)
		fmt.Printf("Jobs in queue: %d\n", size)

	default:
		fmt.Println("Usage:")
		flag.PrintDefaults()
		fmt.Println("\nTo run the Asynq consumer: go run ./cmd/worker")
	}
}
