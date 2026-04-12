package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"app/internal/config"
	"app/pkg/logger"
	"app/pkg/queue"
)

// QueueService manages queue workers and jobs using application config (no global viper).
type QueueService struct {
	cfg     *config.Config
	manager *queue.Manager
	workers map[string]*queue.Worker
	mu      sync.RWMutex
}

// NewQueueService builds a queue service from loaded config.
func NewQueueService(cfg *config.Config) (*QueueService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	qc := queue.Config{
		Driver:  cfg.Queue.Driver,
		Options: make(map[string]any),
	}

	switch cfg.Queue.Driver {
	case "redis":
		redisPort := cfg.Redis.Port
		if redisPort == "" {
			redisPort = "6379"
		}
		connectionStr := fmt.Sprintf("redis://%s:%s/%d", cfg.Redis.Host, redisPort, cfg.Redis.DB)
		if cfg.Redis.Password != "" {
			connectionStr = fmt.Sprintf("redis://:%s@%s:%s/%d", cfg.Redis.Password, cfg.Redis.Host, redisPort, cfg.Redis.DB)
		}
		qc.Options["connection"] = connectionStr
		qc.Options["queue"] = cfg.Queue.Queue
		if cfg.Queue.StreamGroup != "" {
			qc.Options["stream_group"] = cfg.Queue.StreamGroup
		}
		if cfg.Queue.UniqueTTL > 0 {
			qc.Options["unique_ttl"] = time.Duration(cfg.Queue.UniqueTTL) * time.Second
		}

	case "database", "mysql", "postgres", "postgresql", "pg":
		return nil, fmt.Errorf("database driver requires external database connection setup")

	default:
		return nil, fmt.Errorf("unsupported queue driver: %s", cfg.Queue.Driver)
	}

	manager, err := queue.NewManager(qc)
	if err != nil {
		return nil, fmt.Errorf("failed to create queue manager: %v", err)
	}

	return &QueueService{
		cfg:     cfg,
		manager: manager,
		workers: make(map[string]*queue.Worker),
	}, nil
}

// Start launches workers according to cfg.Queue.Queues.
func (s *QueueService) Start() error {
	if len(s.cfg.Queue.Queues) == 0 {
		return fmt.Errorf("no queues configured")
	}

	wcfg := s.cfg.Queue.Worker
	for name, qd := range s.cfg.Queue.Queues {
		processes := qd.Processes
		if processes < 1 {
			processes = 1
		}

		options := queue.WorkerOptions{
			Sleep:   time.Duration(wcfg.Sleep) * time.Second,
			MaxJobs: int64(wcfg.MaxJobs),
			MaxTime: time.Duration(wcfg.MaxTime) * time.Second,
			Rest:    time.Duration(wcfg.Rest) * time.Second,
			Memory:  int64(wcfg.Memory),
			Tries:   wcfg.Tries,
			Timeout: time.Duration(wcfg.Timeout) * time.Second,
		}

		for i := 0; i < processes; i++ {
			workerName := fmt.Sprintf("%s-%d", name, i+1)
			opts := options
			if s.cfg.Queue.Driver == "redis" {
				opts.ConsumerName = workerName
			}
			worker := queue.NewWorker(s.manager, []string{name}, opts)

			s.mu.Lock()
			s.workers[workerName] = worker
			s.mu.Unlock()

			go func(w *queue.Worker, wname string) {
				logger.Sugared().Infow("queue worker started", "worker", wname)
				w.Start()
				logger.Sugared().Infow("queue worker stopped", "worker", wname)

				s.mu.Lock()
				delete(s.workers, wname)
				s.mu.Unlock()
			}(worker, workerName)
		}
	}

	return nil
}

// Stop stops all workers.
func (s *QueueService) Stop() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for name, worker := range s.workers {
		logger.Sugared().Infow("stopping queue worker", "worker", name)
		worker.Stop()
	}
}

// Push pushes a job to the queue.
func (s *QueueService) Push(ctx context.Context, job queue.JobInterface) error {
	return s.manager.Push(ctx, job)
}

// PushRaw pushes raw payload to a queue.
func (s *QueueService) PushRaw(ctx context.Context, q string, payload []byte, options map[string]interface{}) error {
	return s.manager.PushRaw(ctx, q, payload, options)
}

// Later schedules a delayed job.
func (s *QueueService) Later(ctx context.Context, job queue.JobInterface, delay time.Duration) error {
	return s.manager.Later(ctx, job, delay)
}

// Pop pops a job from the queue.
func (s *QueueService) Pop(ctx context.Context, q string) (queue.JobInterface, error) {
	return s.manager.Pop(ctx, q)
}

// Size returns queue length.
func (s *QueueService) Size(ctx context.Context, q string) (int64, error) {
	return s.manager.Size(ctx, q)
}

// Delete removes a job.
func (s *QueueService) Delete(ctx context.Context, q string, job queue.JobInterface) error {
	return s.manager.Delete(ctx, q, job)
}

// Release returns a job to the queue after delay.
func (s *QueueService) Release(ctx context.Context, q string, job queue.JobInterface, delay time.Duration) error {
	return s.manager.Release(ctx, q, job, delay)
}

// Clear empties a queue.
func (s *QueueService) Clear(ctx context.Context, q string) error {
	return s.manager.Clear(ctx, q)
}

// GetWorkerCount returns running worker count.
func (s *QueueService) GetWorkerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.workers)
}

// GetActiveQueues returns worker instance names currently registered.
func (s *QueueService) GetActiveQueues() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.workers))
	for name := range s.workers {
		out = append(out, name)
	}
	return out
}
