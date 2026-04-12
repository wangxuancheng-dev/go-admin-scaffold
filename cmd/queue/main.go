package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"app/internal/config"
	"app/internal/core/services"
)

var (
	// 命令行参数
	configFile  string
	queueName   string
	listQueues  bool
	clearQueue  bool
	stopQueue   bool
	startQueue  bool
	statusQueue bool
)

func init() {
	// 注册命令行参数
	flag.StringVar(&configFile, "config", "configs/config.yaml", "配置文件路径")
	flag.StringVar(&queueName, "queue", "", "队列名称")
	flag.BoolVar(&listQueues, "list", false, "列出所有队列")
	flag.BoolVar(&clearQueue, "clear", false, "清空队列")
	flag.BoolVar(&stopQueue, "stop", false, "停止队列")
	flag.BoolVar(&startQueue, "start", false, "启动队列")
	flag.BoolVar(&statusQueue, "status", false, "查询队列状态")
}

func main() {
	// 解析命令行参数
	flag.Parse()

	cfg, err := config.LoadConfigFromFile(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	queueService, err := services.NewQueueService(cfg)
	if err != nil {
		log.Fatalf("Failed to create queue service: %v", err)
	}

	// 处理命令
	switch {
	case listQueues:
		// 列出所有队列
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
		// 清空队列
		if queueName == "" {
			log.Fatal("Queue name is required")
		}

		ctx := context.Background()
		if err := queueService.Clear(ctx, queueName); err != nil {
			log.Fatalf("Failed to clear queue %s: %v", queueName, err)
		}
		fmt.Printf("Queue %s cleared\n", queueName)

	case stopQueue:
		// 停止队列
		if queueName == "" {
			// 停止所有队列
			queueService.Stop()
			fmt.Println("All queues stopped")
			return
		}

		// TODO: 实现停止指定队列的功能
		fmt.Printf("Stopping queue %s...\n", queueName)

	case startQueue:
		// 启动队列
		if err := queueService.Start(); err != nil {
			log.Fatalf("Failed to start queue service: %v", err)
		}

		// 等待信号
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		fmt.Println("Queue service started, press Ctrl+C to stop")
		<-sigChan

		// 停止服务
		queueService.Stop()
		fmt.Println("Queue service stopped")

	case statusQueue:
		// 查询队列状态
		if queueName == "" {
			// 如果没有指定队列名称，显示所有队列的状态
			queues := queueService.GetActiveQueues()
			if len(queues) == 0 {
				fmt.Println("No active queues")
				return
			}

			fmt.Println("Queue Status:")
			fmt.Println("=============")
			ctx := context.Background()
			for _, name := range queues {
				size, err := queueService.Size(ctx, name)
				if err != nil {
					fmt.Printf("%-20s: Error - %v\n", name, err)
				} else {
					fmt.Printf("%-20s: %d jobs\n", name, size)
				}
			}
		} else {
			// 查询指定队列的状态
			ctx := context.Background()
			size, err := queueService.Size(ctx, queueName)
			if err != nil {
				log.Fatalf("Failed to get queue size for %s: %v", queueName, err)
			}

			fmt.Printf("Queue: %s\n", queueName)
			fmt.Printf("Jobs in queue: %d\n", size)

			// 获取工作进程数量
			workerCount := queueService.GetWorkerCount()
			fmt.Printf("Active workers: %d\n", workerCount)
		}

	default:
		// 显示帮助信息
		fmt.Println("Usage:")
		flag.PrintDefaults()
	}
}

