package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"app/internal/config"
	"app/internal/core/jobs" // RegisterJobType (init) + RegisterAsynqHandlers
	"app/pkg/logger"
	"app/pkg/queue"

	"github.com/hibiken/asynq"
)

// QueueService manages Asynq client (enqueue) and Server (workers) using application config.
type QueueService struct {
	cfg         *config.Config
	manager     *queue.Manager
	asynqSrv    *asynq.Server
	asynqStop   chan struct{} // closed by Stop() so the server goroutine can Shutdown (must not use Run+external Shutdown: waitForSignals deadlock)
	asynqWG     sync.WaitGroup
	concurrency int
	mu          sync.RWMutex
	started     bool
}

// NewQueueService builds a queue service from loaded config.
func NewQueueService(cfg *config.Config) (*QueueService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	driver := strings.ToLower(strings.TrimSpace(cfg.Queue.Driver))
	if driver == "" {
		driver = "redis"
	}
	if driver != "redis" && driver != "asynq" {
		return nil, fmt.Errorf("unsupported queue driver %q (only redis/asynq)", cfg.Queue.Driver)
	}

	connectionStr := config.QueueRedisConnectionURL(cfg)
	qc := queue.Config{
		Driver:  "redis",
		Options: map[string]any{},
	}
	qc.Options["connection"] = connectionStr
	qc.Options["queue"] = cfg.Queue.Queue
	if cfg.Queue.UniqueTTL > 0 {
		qc.Options["unique_ttl"] = time.Duration(cfg.Queue.UniqueTTL) * time.Second
	}

	manager, err := queue.NewManager(qc)
	if err != nil {
		return nil, fmt.Errorf("failed to create queue manager: %v", err)
	}

	return &QueueService{
		cfg:     cfg,
		manager: manager,
	}, nil
}

// Start launches a single asynq.Server for all configured queues (weights from priority, concurrency from sum of processes).
func (s *QueueService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return fmt.Errorf("queue service already started")
	}
	if len(s.cfg.Queue.Queues) == 0 {
		return fmt.Errorf("no queues configured")
	}

	redisOpt, err := queue.AsynqRedisConnOpt(config.QueueRedisConnectionURL(s.cfg))
	if err != nil {
		return fmt.Errorf("asynq redis opt: %w", err)
	}

	queues := make(map[string]int)
	concurrency := 0
	for name, qd := range s.cfg.Queue.Queues {
		p := qd.Priority
		if p <= 0 {
			p = 1
		}
		queues[name] = p
		proc := qd.Processes
		if proc < 1 {
			proc = 1
		}
		concurrency += proc
	}
	if concurrency <= 0 {
		concurrency = 1
	}

	mux := asynq.NewServeMux()
	jobs.RegisterAsynqHandlers(mux)

	shutdownSec := s.cfg.Queue.Worker.Timeout
	if shutdownSec <= 0 {
		shutdownSec = 30
	}
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency:     concurrency,
		Queues:          queues,
		ShutdownTimeout: time.Duration(shutdownSec) * time.Second,
	})

	s.asynqSrv = srv
	stopCh := make(chan struct{})
	s.asynqStop = stopCh
	s.concurrency = concurrency
	s.started = true

	s.asynqWG.Add(1)
	go func() {
		defer s.asynqWG.Done()
		logger.Sugared().Infow("asynq server starting", "concurrency", concurrency, "queues", queues)
		if err := srv.Start(mux); err != nil {
			logger.Sugared().Errorw("asynq server start failed", "error", err)
			s.mu.Lock()
			if s.asynqSrv == srv {
				s.started = false
				s.asynqSrv = nil
				s.asynqStop = nil
			}
			s.mu.Unlock()
			return
		}
		logger.Sugared().Infow("asynq server started", "concurrency", concurrency, "queues", queues)
		<-stopCh
		srv.Shutdown()
	}()

	return nil
}

// Stop shuts down the asynq Server.
func (s *QueueService) Stop() {
	s.mu.Lock()
	srv := s.asynqSrv
	stopCh := s.asynqStop
	s.asynqSrv = nil
	s.asynqStop = nil
	s.started = false
	s.mu.Unlock()

	if srv != nil && stopCh != nil {
		logger.Sugared().Info("stopping asynq server")
		close(stopCh)
		s.asynqWG.Wait()
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

// GetWorkerCount returns asynq concurrency when the server is running, else 0.
func (s *QueueService) GetWorkerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.started {
		return 0
	}
	return s.concurrency
}

// GetActiveQueues returns configured queue names (sorted).
func (s *QueueService) GetActiveQueues() []string {
	out := make([]string, 0, len(s.cfg.Queue.Queues))
	for name := range s.cfg.Queue.Queues {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
