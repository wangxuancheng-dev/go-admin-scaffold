package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"app/internal/config"
	"app/pkg/database"
	"app/pkg/queue"
)

var (
	queueName string
	driver    string
	showAll   bool
)

func init() {
	flag.StringVar(&queueName, "queue", "", "队列名称")
	flag.StringVar(&driver, "driver", "", "队列驱动 (redis, database)")
	flag.BoolVar(&showAll, "all", false, "显示所有队列状态")
}

func main() {
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if showAll {
		// 显示所有队列状态
		fmt.Println("队列状态概览:")
		fmt.Println("=============")

		// 显示Redis队列状态
		fmt.Println("\nRedis 队列:")
		fmt.Println("-----------")
		redisQueues := []string{"default", "low"}
		for _, qName := range redisQueues {
			size, err := getRedisQueueSize(cfg, qName)
			if err != nil {
				fmt.Printf("%-15s: Error - %v\n", qName, err)
			} else {
				fmt.Printf("%-15s: %d jobs\n", qName, size)
			}
		}

		// 显示Database队列状态
		fmt.Println("\nDatabase 队列:")
		fmt.Println("--------------")
		dbQueues := []string{"high"}
		for _, qName := range dbQueues {
			size, err := getDatabaseQueueSize(cfg, qName)
			if err != nil {
				fmt.Printf("%-15s: Error - %v\n", qName, err)
			} else {
				fmt.Printf("%-15s: %d jobs\n", qName, size)
			}
		}

	} else if queueName != "" {
		// 查询指定队列
		if driver == "" {
			// 尝试两种驱动
			fmt.Printf("队列 '%s' 状态:\n", queueName)
			fmt.Println("=============")

			// 尝试Redis
			size, err := getRedisQueueSize(cfg, queueName)
			if err == nil {
				fmt.Printf("Redis 驱动: %d jobs\n", size)
			} else {
				fmt.Printf("Redis 驱动: Error - %v\n", err)
			}

			// 尝试Database
			size, err = getDatabaseQueueSize(cfg, queueName)
			if err == nil {
				fmt.Printf("Database 驱动: %d jobs\n", size)
			} else {
				fmt.Printf("Database 驱动: Error - %v\n", err)
			}

		} else {
			// 使用指定驱动
			var size int64
			var err error

			switch driver {
			case "redis":
				size, err = getRedisQueueSize(cfg, queueName)
			case "database":
				size, err = getDatabaseQueueSize(cfg, queueName)
			default:
				log.Fatalf("Unsupported driver: %s", driver)
			}

			if err != nil {
				log.Fatalf("Failed to get queue size: %v", err)
			}

			fmt.Printf("队列 '%s' (%s 驱动): %d jobs\n", queueName, driver, size)
		}

	} else {
		// 显示帮助
		fmt.Println("队列状态查询工具")
		fmt.Println("================")
		fmt.Println("使用方法:")
		fmt.Println("  -all                   显示所有队列状态")
		fmt.Println("  -queue=<name>          查询指定队列")
		fmt.Println("  -driver=<redis|database> 指定驱动类型")
		fmt.Println("")
		fmt.Println("示例:")
		fmt.Println("  ./queue-status -all")
		fmt.Println("  ./queue-status -queue=default")
		fmt.Println("  ./queue-status -queue=high -driver=database")
	}
}

func getRedisQueueSize(cfg *config.Config, queueName string) (int64, error) {
	// 创建Redis队列管理器
	queueConfig := queue.Config{
		Driver: "redis",
		Options: map[string]interface{}{
			"connection": fmt.Sprintf("redis://%s:%s/%d", cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB),
			"queue":      queueName,
		},
	}

	manager, err := queue.NewManager(queueConfig)
	if err != nil {
		return 0, err
	}
	defer manager.Close()

	return manager.Size(context.Background(), queueName)
}

func getDatabaseQueueSize(cfg *config.Config, queueName string) (int64, error) {
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
		return 0, err
	}

	// 创建Database队列管理器
	queueConfig := queue.Config{
		Driver: "database",
		Options: map[string]interface{}{
			"db":    db,
			"queue": queueName,
		},
	}

	manager, err := queue.NewManager(queueConfig)
	if err != nil {
		return 0, err
	}
	defer manager.Close()

	return manager.Size(context.Background(), queueName)
}
