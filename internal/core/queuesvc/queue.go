package queuesvc

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go-admin-scaffold/internal/config"
	_ "go-admin-scaffold/internal/core/jobs" // RegisterJobType via init
	"go-admin-scaffold/pkg/queue"
)

// QueueService manages Asynq enqueue / inspect / clear using application config.
// Run consumers with cmd/worker — this type does not embed an Asynq Server.
type QueueService struct {
	cfg     *config.Config
	manager *queue.Manager
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

// GetActiveQueues returns configured queue names (sorted).
func (s *QueueService) GetActiveQueues() []string {
	out := make([]string, 0, len(s.cfg.Queue.Queues))
	for name := range s.cfg.Queue.Queues {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
