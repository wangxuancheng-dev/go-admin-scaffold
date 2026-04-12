package schedule

import (
	"context"
	"fmt"
	"sync"
	"time"

	"app/pkg/console"
	"app/pkg/locker"
	"app/pkg/logger"

	"github.com/robfig/cron/v3"
)

// Scheduler manages all scheduled tasks
type Scheduler struct {
	cron     *cron.Cron
	tasks    []Task
	manager  *console.Manager
	mu       sync.RWMutex
	location *time.Location
	locker   *locker.RedisLocker
}

// Task represents a scheduled task
type Task struct {
	Name     string
	Schedule string
	Command  console.Command
	Unique   bool // Whether the task should run on only one server
}

// NewScheduler creates a new scheduler instance
func NewScheduler(manager *console.Manager, locker *locker.RedisLocker) *Scheduler {
	loc, _ := time.LoadLocation("Local")
	return &Scheduler{
		cron:     cron.New(cron.WithLocation(loc)),
		tasks:    make([]Task, 0),
		manager:  manager,
		location: loc,
		locker:   locker,
	}
}

// Task creates a new task
func (s *Scheduler) Task(name string, command console.Command) *TaskBuilder {
	return &TaskBuilder{
		scheduler: s,
		task: Task{
			Name:    name,
			Command: command,
			Unique:  false, // Default to non-unique
		},
	}
}

// Command creates a new task from command name
func (s *Scheduler) Command(name string, args ...string) *TaskBuilder {
	cmd := s.manager.FindCommand(name)
	if cmd == nil {
		return nil
	}
	return s.Task(name, cmd)
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, task := range s.tasks {
		task := task // Create new variable for closure
		_, err := s.cron.AddFunc(task.Schedule, func() {
			if err := s.runTask(ctx, task); err != nil {
				logger.Sugared().Errorw("scheduler task failed", "task", task.Name, "error", err)
			}
		})
		if err != nil {
			return fmt.Errorf("failed to add task %s: %v", task.Name, err)
		}
	}

	s.cron.Start()
	return nil
}

// runTask runs a single task with distributed lock if needed
func (s *Scheduler) runTask(ctx context.Context, task Task) error {
	if !task.Unique {
		// For non-unique tasks, run directly
		return task.Command.Handle(ctx)
	}

	// For unique tasks, try to acquire a lock first
	lockKey := fmt.Sprintf("scheduler:lock:%s", task.Name)
	// Set TTL to slightly longer than the expected task duration
	// Adjust the TTL based on your needs
	ttl := 30 * time.Minute

	acquired, err := s.locker.TryLock(ctx, lockKey, ttl)
	if err != nil {
		return fmt.Errorf("failed to acquire lock for task %s: %v", task.Name, err)
	}

	if !acquired {
		logger.Sugared().Infow("scheduler task skipped, lock held elsewhere", "task", task.Name)
		return nil
	}

	// Run the task and ensure we release the lock afterward
	defer func() {
		if err := s.locker.Unlock(ctx, lockKey); err != nil {
			logger.Sugared().Warnw("scheduler lock release failed", "task", task.Name, "error", err)
		}
	}()

	return task.Command.Handle(ctx)
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cron != nil {
		s.cron.Stop()
	}
}

// AddTask adds a task to the scheduler
func (s *Scheduler) AddTask(task Task) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks = append(s.tasks, task)
}

// TaskBuilder helps build a task with fluent interface
type TaskBuilder struct {
	scheduler *Scheduler
	task      Task
}

// Unique marks the task as unique (should only run on one server)
func (b *TaskBuilder) Unique() *TaskBuilder {
	b.task.Unique = true
	return b
}

// Cron sets a custom cron schedule
func (b *TaskBuilder) Cron(schedule string) *TaskBuilder {
	b.task.Schedule = schedule
	return b
}

// EveryMinute sets the task to run every minute
func (b *TaskBuilder) EveryMinute() *TaskBuilder {
	return b.Cron("* * * * *")
}

// EveryFiveMinutes sets the task to run every five minutes
func (b *TaskBuilder) EveryFiveMinutes() *TaskBuilder {
	return b.Cron("*/5 * * * *")
}

// EveryTenMinutes sets the task to run every ten minutes
func (b *TaskBuilder) EveryTenMinutes() *TaskBuilder {
	return b.Cron("*/10 * * * *")
}

// EveryThirtyMinutes sets the task to run every thirty minutes
func (b *TaskBuilder) EveryThirtyMinutes() *TaskBuilder {
	return b.Cron("*/30 * * * *")
}

// Hourly sets the task to run hourly
func (b *TaskBuilder) Hourly() *TaskBuilder {
	return b.Cron("0 * * * *")
}

// Daily sets the task to run daily
func (b *TaskBuilder) Daily() *TaskBuilder {
	return b.Cron("0 0 * * *")
}

// Weekly sets the task to run weekly
func (b *TaskBuilder) Weekly() *TaskBuilder {
	return b.Cron("0 0 * * 0")
}

// Monthly sets the task to run monthly
func (b *TaskBuilder) Monthly() *TaskBuilder {
	return b.Cron("0 0 1 * *")
}

// At sets a specific time for daily tasks
func (b *TaskBuilder) At(time string) *TaskBuilder {
	t, err := b.scheduler.parseTime(time)
	if err != nil {
		return b
	}
	b.task.Schedule = fmt.Sprintf("%d %d * * *", t.Minute(), t.Hour())
	return b
}

// Register registers the task with the scheduler
func (b *TaskBuilder) Register() {
	b.scheduler.AddTask(b.task)
}

// parseTime parses time string in HH:mm format
func (s *Scheduler) parseTime(timeStr string) (time.Time, error) {
	return time.ParseInLocation("15:04", timeStr, s.location)
}
