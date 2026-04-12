package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"app/internal/config"
	"app/pkg/queue"
)

var (
	queueName string
	showAll   bool
)

func init() {
	flag.StringVar(&queueName, "queue", "", "队列名称")
	flag.BoolVar(&showAll, "all", false, "显示所有队列状态")
}

func main() {
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if showAll {
		fmt.Println("队列状态概览 (Asynq / Redis):")
		fmt.Println("=====================")
		for _, qName := range []string{"default", "high", "low"} {
			size, err := getQueueSize(cfg, qName)
			if err != nil {
				fmt.Printf("%-15s: Error - %v\n", qName, err)
			} else {
				fmt.Printf("%-15s: %d tasks (total)\n", qName, size)
			}
		}
		return
	}

	if queueName != "" {
		size, err := getQueueSize(cfg, queueName)
		if err != nil {
			log.Fatalf("Failed to get queue size: %v", err)
		}
		fmt.Printf("队列 '%s': %d tasks (pending+scheduled+active+…)\n", queueName, size)
		return
	}

	fmt.Println("队列状态查询工具 (Asynq)")
	fmt.Println("========================")
	fmt.Println("  -all            显示 default / high / low 任务总数")
	fmt.Println("  -queue=<name>   查询指定队列")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  ./queue-status -all")
	fmt.Println("  ./queue-status -queue=default")
}

func getQueueSize(cfg *config.Config, name string) (int64, error) {
	queueConfig := queue.Config{
		Driver: "redis",
		Options: map[string]interface{}{
			"connection": config.QueueRedisConnectionURL(cfg),
			"queue":      name,
		},
	}

	manager, err := queue.NewManager(queueConfig)
	if err != nil {
		return 0, err
	}
	defer manager.Close()

	return manager.Size(context.Background(), name)
}
