package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"app/internal/config"
	"app/internal/core/jobs"
	"app/pkg/queue"

	"github.com/hibiken/asynq"
)

func main() {
	seed := flag.Bool("seed", false, "向 default / high / low 写入示例任务（供 worker 联调），不跑自动化用例")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *seed {
		seedDemoQueues(cfg)
		return
	}

	fmt.Println("=== 测试 Asynq 队列（自动化） ===")
	queueConfig := queue.Config{
		Driver: "redis",
		Options: map[string]any{
			"connection": config.QueueRedisConnectionURL(cfg),
			"queue":      "test-asynq",
		},
	}

	manager, err := queue.NewManager(queueConfig)
	if err != nil {
		log.Fatalf("Failed to create queue manager: %v", err)
	}
	defer manager.Close()

	testQueue(manager, cfg)
}

// asynqWorkBacklog 为尚待调度/执行的任务（不含 completed/archived，避免 Size 含已完成导致永远非 0）
func asynqWorkBacklog(info *asynq.QueueInfo) int {
	if info == nil {
		return 0
	}
	return info.Pending + info.Active + info.Scheduled + info.Retry + info.Aggregating
}

// testConsumeDrain 启动临时 Asynq Server，直到队列无积压工作（与 pkg/queue 中 queue-not-exist 判定一致）
func testConsumeDrain(ctx context.Context, cfg *config.Config, qname string, maxWait time.Duration) error {
	redisOpt, err := queue.AsynqRedisConnOpt(config.QueueRedisConnectionURL(cfg))
	if err != nil {
		return err
	}
	insp := asynq.NewInspector(redisOpt)
	defer insp.Close()

	mux := asynq.NewServeMux()
	jobs.RegisterAsynqHandlers(mux)

	shutdownSec := cfg.Queue.Worker.Timeout
	if shutdownSec <= 0 {
		shutdownSec = 30
	}
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency:     2,
		Queues:          map[string]int{qname: 1},
		ShutdownTimeout: time.Duration(shutdownSec) * time.Second,
	})
	if err := srv.Start(mux); err != nil {
		return fmt.Errorf("asynq Start: %w", err)
	}
	defer srv.Shutdown()

	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := insp.GetQueueInfo(qname)
		if err != nil {
			if errors.Is(err, asynq.ErrQueueNotFound) {
				return nil
			}
			if strings.Contains(err.Error(), "does not exist") && strings.Contains(err.Error(), "queue") {
				return nil
			}
			return err
		}
		if asynqWorkBacklog(info) == 0 {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	info, _ := insp.GetQueueInfo(qname)
	return fmt.Errorf("timeout after %v, backlog still %d", maxWait, asynqWorkBacklog(info))
}

// seedDemoQueues 原 add-test-jobs：往业务队列灌数据，便于手动起 worker 验证。
func seedDemoQueues(cfg *config.Config) {
	redisURL := config.QueueRedisConnectionURL(cfg)
	ctx := context.Background()

	defaultMgr, err := queue.NewManager(queue.Config{
		Driver: "redis",
		Options: map[string]any{"connection": redisURL, "queue": "default"},
	})
	if err != nil {
		log.Fatalf("redis default: %v", err)
	}
	defer defaultMgr.Close()

	highMgr, err := queue.NewManager(queue.Config{
		Driver: "redis",
		Options: map[string]any{"connection": redisURL, "queue": "high"},
	})
	if err != nil {
		log.Fatalf("redis high: %v", err)
	}
	defer highMgr.Close()

	lowMgr, err := queue.NewManager(queue.Config{
		Driver: "redis",
		Options: map[string]any{"connection": redisURL, "queue": "low"},
	})
	if err != nil {
		log.Fatalf("redis low: %v", err)
	}
	defer lowMgr.Close()

	fmt.Println("=== 写入示例任务（-seed）===")
	fmt.Println("正在添加...")

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
			Message: fmt.Sprintf("default task #%d", i),
		}
		if err := defaultMgr.Push(ctx, job); err != nil {
			log.Printf("push default: %v", err)
		}
	}

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
		if err := highMgr.Push(ctx, job); err != nil {
			log.Printf("push high: %v", err)
		}
	}

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
		if err := lowMgr.Later(ctx, job, time.Duration(i*10)*time.Second); err != nil {
			log.Printf("push low delayed: %v", err)
		}
	}

	fmt.Println("✅ 完成: default×5, high×3, low 延迟×2")
}

func testQueue(manager *queue.Manager, cfg *config.Config) {
	ctx := context.Background()
	queueName := "test-asynq"

	fmt.Println("开始测试 Asynq 驱动...")

	err := manager.Clear(ctx, queueName)
	if err != nil {
		log.Printf("Failed to clear queue: %v", err)
		return
	}
	fmt.Println("✓ 清空队列成功")

	// 1. Push + Size
	fmt.Println("\n1. 测试基本入队与长度:")
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
		Message: "Hello from Asynq queue!",
	}
	if err := manager.Push(ctx, job); err != nil {
		log.Printf("Failed to push job: %v", err)
		return
	}
	fmt.Println("✓ 推送任务成功")

	size, err := manager.Size(ctx, queueName)
	if err != nil {
		log.Printf("Failed to get queue size: %v", err)
		return
	}
	if size < 1 {
		log.Printf("expected size >= 1, got %d", size)
		return
	}
	fmt.Printf("✓ 队列任务数: %d（第 5 步将本进程消费；日常可单独起 worker）\n", size)

	_, err = manager.Pop(ctx, queueName)
	if !errors.Is(err, queue.ErrPullNotSupported) {
		log.Printf("expected ErrPullNotSupported from Pop, got %v", err)
		return
	}
	fmt.Println("✓ Pop 正确返回 ErrPullNotSupported（Asynq 由 Server 消费）")

	// 2. 延迟入队
	fmt.Println("\n2. 测试延迟任务 (Later):")
	if err := manager.Clear(ctx, queueName); err != nil {
		log.Printf("clear: %v", err)
		return
	}
	delayedJob := &jobs.ExampleJob{
		BaseJob: queue.BaseJob{
			Queue:       queueName,
			Attempts:    0,
			MaxAttempts: 3,
			Delay:       0,
			Timeout:     60 * time.Second,
			RetryAfter:  60 * time.Second,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		Message: "Delayed via Later",
	}
	laterDelay := 3 * time.Second
	if err := manager.Later(ctx, delayedJob, laterDelay); err != nil {
		log.Printf("Later: %v", err)
		return
	}
	fmt.Printf("✓ Later 入队成功（延迟 %v）\n", laterDelay)

	redisOpt, err := queue.AsynqRedisConnOpt(config.QueueRedisConnectionURL(cfg))
	if err != nil {
		log.Printf("asynq redis opt: %v", err)
		return
	}
	inspLater := asynq.NewInspector(redisOpt)
	defer inspLater.Close()

	info0, err := inspLater.GetQueueInfo(queueName)
	if err != nil {
		log.Printf("GetQueueInfo: %v", err)
		return
	}
	if info0.Scheduled < 1 {
		log.Printf("延迟任务应先进 Scheduled，当前 scheduled=%d pending=%d active=%d（若全为 0 则未按延迟入队）",
			info0.Scheduled, info0.Pending, info0.Active)
		return
	}
	fmt.Printf("✓ 入队后立即在 Scheduled（尚未到点执行）: scheduled=%d pending=%d\n",
		info0.Scheduled, info0.Pending)

	// 用 NextProcessAt 校验「真的是延迟到未来某时刻」，而不是误判为立即执行
	scheduledTasks, err := inspLater.ListScheduledTasks(queueName)
	if err != nil {
		log.Printf("ListScheduledTasks: %v", err)
		return
	}
	if len(scheduledTasks) < 1 {
		log.Printf("ListScheduledTasks: 期望至少 1 条 scheduled 任务")
		return
	}
	now := time.Now()
	np := scheduledTasks[0].NextProcessAt
	minAt := now.Add(laterDelay - time.Second)
	maxAt := now.Add(laterDelay + 2 * time.Second)
	if np.Before(minAt) || np.After(maxAt) {
		log.Printf("NextProcessAt=%v 不在期望区间 [%v, %v]（now=%v, delay=%v），延迟可能未生效",
			np, minAt, maxAt, now, laterDelay)
		return
	}
	fmt.Printf("✓ Scheduled 任务 NextProcessAt=%v（相对当前约 %v 后，校验延迟已写入）\n",
		np.Format(time.RFC3339), np.Sub(now).Round(time.Millisecond))

	// 无 asynq.Server 时 forwarder 不跑，Scheduled 永远不会进 Pending；须起临时 Server 验证到点转发与执行
	fmt.Println("✓ 启动临时 Server（内置 forwarder 会按间隔扫描 Scheduled，默认约每 5s）…")
	if err := testConsumeDrain(ctx, cfg, queueName, laterDelay+25*time.Second); err != nil {
		log.Printf("延迟任务在临时 Server 下 drain: %v", err)
		return
	}
	fmt.Println("✓ 延迟任务已由临时 Server 转发并消费（Scheduled→Pending→执行，积压为 0）")

	// 3. Raw
	fmt.Println("\n3. 测试 PushRaw (task_type=raw):")
	rawData := map[string]interface{}{
		"type":    "email",
		"to":      "test@example.com",
		"subject": "Test Email",
		"body":    "This is a test from queue-test",
	}
	rawPayload, _ := json.Marshal(rawData)
	if err := manager.PushRaw(ctx, queueName, rawPayload, map[string]interface{}{
		"task_type":    "raw",
		"delay":        2 * time.Second,
		"max_attempts": 3,
		"timeout":      30 * time.Second,
	}); err != nil {
		log.Printf("PushRaw: %v", err)
		return
	}
	fmt.Println("✓ PushRaw 成功")

	// 4. UniqueKey
	fmt.Println("\n4. 测试 UniqueKey:")
	uniqueKey := "queue-test-uniq-asynq"
	uniqueJob := &jobs.ExampleJob{
		BaseJob: queue.BaseJob{
			Queue:       queueName,
			UniqueKey:   uniqueKey,
			Attempts:    0,
			MaxAttempts: 3,
			Delay:       0,
			Timeout:     60 * time.Second,
			RetryAfter:  60 * time.Second,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		Message: "unique first",
	}
	if err := manager.Push(ctx, uniqueJob); err != nil {
		log.Printf("push unique: %v", err)
		return
	}
	fmt.Printf("✓ 首次入队 (UniqueKey=%s)\n", uniqueKey)

	dupJob := &jobs.ExampleJob{
		BaseJob: queue.BaseJob{
			Queue:       queueName,
			UniqueKey:   uniqueKey,
			Attempts:    0,
			MaxAttempts: 3,
			Delay:       0,
			Timeout:     60 * time.Second,
			RetryAfter:  60 * time.Second,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		Message: "duplicate should fail",
	}
	err = manager.Push(ctx, dupJob)
	if err == nil {
		log.Printf("Expected duplicate push to return ErrDuplicateJob")
		return
	}
	if !errors.Is(err, queue.ErrDuplicateJob) {
		log.Printf("Expected ErrDuplicateJob, got: %v", err)
		return
	}
	fmt.Println("✓ 相同 UniqueKey 再次入队被拒绝 (ErrDuplicateJob)")

	if err := manager.Clear(ctx, queueName); err != nil {
		log.Printf("clear before re-push: %v", err)
		return
	}
	if err := manager.Push(ctx, dupJob); err != nil {
		log.Printf("Push after clear should succeed: %v", err)
		return
	}
	fmt.Println("✓ 清空后可再次使用相同 UniqueKey 入队")

	if err := manager.Clear(ctx, queueName); err != nil {
		log.Printf("final clear: %v", err)
		return
	}
	fmt.Println("✓ 最终清空队列成功")

	// 5. 本进程消费（raw 任务立即 Handle，无需等 ExampleJob 的 Sleep）
	fmt.Println("\n5. 测试本进程 Asynq Server 消费:")
	if err := manager.Clear(ctx, queueName); err != nil {
		log.Printf("clear before consume: %v", err)
		return
	}
	rawConsume, _ := json.Marshal(map[string]any{"step": "consume-test"})
	if err := manager.PushRaw(ctx, queueName, rawConsume, map[string]interface{}{
		"task_type":    "raw",
		"max_attempts": 2,
		"timeout":      30 * time.Second,
	}); err != nil {
		log.Printf("PushRaw for consume: %v", err)
		return
	}
	fmt.Println("✓ 已入队 raw 任务，启动临时 Server…")
	if err := testConsumeDrain(ctx, cfg, queueName, 25*time.Second); err != nil {
		log.Printf("consume drain: %v", err)
		return
	}
	fmt.Println("✓ 任务已消费（pending/active/scheduled/retry 积压为 0）")

	if err := manager.Clear(ctx, queueName); err != nil {
		log.Printf("clear after consume: %v", err)
		return
	}
	fmt.Println("✓ 消费后清空队列")

	fmt.Println("\nAsynq 驱动测试完成! ✅")
}
