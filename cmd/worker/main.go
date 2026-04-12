package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"app/internal/config"
	"app/pkg/queue"
)

type Worker struct {
	queue    queue.QueueInterface
	handlers map[string]JobHandler
	stop     chan struct{}
	wg       sync.WaitGroup
}

type JobHandler func(ctx context.Context, payload []byte) error

func NewWorker(q queue.QueueInterface) *Worker {
	return &Worker{
		queue:    q,
		handlers: make(map[string]JobHandler),
		stop:     make(chan struct{}),
	}
}

func (w *Worker) RegisterHandler(queueName string, handler JobHandler) {
	w.handlers[queueName] = handler
}

func (w *Worker) Start(concurrency int) {
	for i := 0; i < concurrency; i++ {
		w.wg.Add(1)
		go w.process()
	}
}

func (w *Worker) Stop() {
	close(w.stop)
	w.wg.Wait()
}

func (w *Worker) process() {
	defer w.wg.Done()

	for {
		select {
		case <-w.stop:
			return
		default:
			for queueName := range w.handlers {
				ctx := context.Background()
				job, err := w.queue.Pop(ctx, queueName)
				if err == queue.ErrQueueEmpty {
					time.Sleep(time.Second)
					continue
				}
				if err != nil {
					log.Printf("Error popping job from queue %s: %v", queueName, err)
					continue
				}

				handler := w.handlers[queueName]
				if err := handler(ctx, job.GetPayload()); err != nil {
					log.Printf("Error processing job %s: %v", job.GetID(), err)
					if job.GetAttempts() < job.GetMaxAttempts() {
						delay := time.Duration(job.GetAttempts()*job.GetAttempts()) * time.Second
						_ = w.queue.Release(ctx, queueName, job, delay)
					} else {
						_ = w.queue.Delete(ctx, queueName, job)
					}
				} else {
					_ = w.queue.Delete(ctx, queueName, job)
				}
			}
		}
	}
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	queueConfig := queue.Config{
		Driver: cfg.Queue.Driver,
	}

	queueConfig.Options = make(map[string]any)
	queueConfig.Options["connection"] = cfg.Queue.Connection.Redis
	queueConfig.Options["queue"] = "default"

	// Create queue instance
	q, err := queue.NewManager(queueConfig)
	if err != nil {
		log.Fatalf("create queue: %v", err)
	}
	defer q.Close()

	// ====================== 这里加了启动日志 ======================
	log.Println("✅ 队列连接成功！Redis:", cfg.Queue.Connection.Redis)
	log.Println("✅ Worker 已启动，等待任务中...")

	// Create worker
	worker := NewWorker(q)

	// Register job handlers
	worker.RegisterHandler("emails", func(ctx context.Context, payload []byte) error {
		log.Println("📩 处理邮件任务:", string(payload))
		return nil
	})

	worker.RegisterHandler("notifications", func(ctx context.Context, payload []byte) error {
		log.Println("🔔 处理通知任务:", string(payload))
		return nil
	})

	// Start worker with concurrency
	worker.Start(5)

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("🛑 正在关闭 Worker...")
	worker.Stop()
	log.Println("✅ Worker 已安全退出")
}
