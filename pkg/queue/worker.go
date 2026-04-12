package queue

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// Worker 队列工作进程
type Worker struct {
	manager *Manager
	queues  []string
	options WorkerOptions
	wg      sync.WaitGroup
	stop    chan struct{}
}

// WorkerOptions 工作进程选项
type WorkerOptions struct {
	// ConsumerName Redis Streams 下 XREADGROUP 的 consumer（多进程需唯一，默认不填则由驱动生成）
	ConsumerName string
	// Sleep 无任务时休眠时间
	Sleep time.Duration
	// MaxJobs 最大处理任务数
	MaxJobs int64
	// MaxTime 最大运行时间
	MaxTime time.Duration
	// Rest 处理完一个任务后休息时间
	Rest time.Duration
	// Memory 内存限制(MB)
	Memory int64
	// Tries 任务最大重试次数
	Tries int
	// Timeout 任务超时时间
	Timeout time.Duration
}

// DefaultWorkerOptions 默认工作进程选项
var DefaultWorkerOptions = WorkerOptions{
	Sleep:   3 * time.Second,
	MaxJobs: 0,
	MaxTime: 0,
	Rest:    0,
	Memory:  128,
	Tries:   3,
	Timeout: 60 * time.Second,
}

// NewWorker 创建队列工作进程
func NewWorker(manager *Manager, queues []string, options WorkerOptions) *Worker {
	if options.Sleep == 0 {
		options.Sleep = DefaultWorkerOptions.Sleep
	}
	if options.Memory == 0 {
		options.Memory = DefaultWorkerOptions.Memory
	}
	if options.Tries == 0 {
		options.Tries = DefaultWorkerOptions.Tries
	}
	if options.Timeout == 0 {
		options.Timeout = DefaultWorkerOptions.Timeout
	}

	return &Worker{
		manager: manager,
		queues:  queues,
		options: options,
		stop:    make(chan struct{}),
	}
}

// Start 启动工作进程
func (w *Worker) Start() {
	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动工作协程
	w.wg.Add(1)
	go w.run()

	// 等待信号
	<-sigChan
	log.Println("Received stop signal, shutting down...")

	// 停止工作进程
	close(w.stop)
	w.wg.Wait()

	log.Println("Worker stopped")
}

// run 运行工作进程
func (w *Worker) run() {
	defer w.wg.Done()

	startTime := time.Now()
	var jobsProcessed int64

	for {
		select {
		case <-w.stop:
			return
		default:
			// 检查是否达到最大任务数
			if w.options.MaxJobs > 0 && jobsProcessed >= w.options.MaxJobs {
				log.Printf("Reached max jobs limit (%d), stopping...", w.options.MaxJobs)
				return
			}

			// 检查是否达到最大运行时间
			if w.options.MaxTime > 0 && time.Since(startTime) >= w.options.MaxTime {
				log.Printf("Reached max time limit (%v), stopping...", w.options.MaxTime)
				return
			}

			// 检查内存使用
			if w.options.Memory > 0 {
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				if m.Alloc/1024/1024 > uint64(w.options.Memory) {
					log.Printf("Memory limit exceeded (%dMB), stopping...", w.options.Memory)
					return
				}
			}

			// 处理任务
			if _, err := w.processNextJob(); err != nil {
				if err == ErrQueueEmpty {
					// 队列为空，休眠
					time.Sleep(w.options.Sleep)
					continue
				}
				log.Printf("Error processing job: %v", err)
				continue
			}

			// 增加处理计数
			jobsProcessed++

			// 处理完一个任务后休息
			if w.options.Rest > 0 {
				time.Sleep(w.options.Rest)
			}
		}
	}
}

// processNextJob 处理下一个任务
func (w *Worker) processNextJob() (JobInterface, error) {
	ctx := context.Background()
	if w.options.ConsumerName != "" {
		ctx = WithRedisConsumer(ctx, w.options.ConsumerName)
	}

	// 遍历所有队列
	for _, queue := range w.queues {
		// 获取任务
		job, err := w.manager.Pop(ctx, queue)
		if err == ErrQueueEmpty {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("error popping job from queue %s: %v", queue, err)
		}

		// 处理任务
		if err := w.processJob(ctx, queue, job); err != nil {
			log.Printf("Error processing job from queue %s: %v", queue, err)
		}

		return job, nil
	}

	return nil, ErrQueueEmpty
}

// processJob 处理任务
func (w *Worker) processJob(ctx context.Context, queue string, job JobInterface) error {
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, w.options.Timeout)
	defer cancel()

	// 创建错误通道
	errChan := make(chan error, 1)

	// 在协程中执行任务
	go func() {
		errChan <- job.Handle()
	}()

	// 等待任务完成或超时
	select {
	case err := <-errChan:
		if err != nil {
			// 任务失败，尝试重试
			if job.GetAttempts() < w.options.Tries {
				delay := w.calculateBackoff(job)
				return w.manager.Release(ctx, queue, job, delay)
			}
			// 超过重试次数，删除任务
			return w.manager.Delete(ctx, queue, job)
		}
		// 任务成功，删除任务
		return w.manager.Delete(ctx, queue, job)
	case <-ctx.Done():
		// 任务超时，尝试重试
		if job.GetAttempts() < w.options.Tries {
			delay := w.calculateBackoff(job)
			return w.manager.Release(ctx, queue, job, delay)
		}
		// 超过重试次数，删除任务
		return w.manager.Delete(ctx, queue, job)
	}
}

// calculateBackoff 计算退避时间
func (w *Worker) calculateBackoff(job JobInterface) time.Duration {
	backoff := job.GetBackoff()
	if len(backoff) > 0 {
		attempt := job.GetAttempts()
		if attempt < len(backoff) {
			return backoff[attempt]
		}
		return backoff[len(backoff)-1]
	}
	return job.GetRetryAfter()
}

// Stop 停止工作进程
func (w *Worker) Stop() {
	close(w.stop)
	w.wg.Wait()
}
