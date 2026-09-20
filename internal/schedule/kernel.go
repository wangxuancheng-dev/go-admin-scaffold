package schedule

import (
	"context"

	"go-admin-scaffold/pkg/logger"
)

// Kernel manages the scheduler
type Kernel struct {
	scheduler *Scheduler
}

// NewKernel creates a new scheduler kernel
func NewKernel(scheduler *Scheduler) *Kernel {
	return &Kernel{
		scheduler: scheduler,
	}
}

// Schedule defines scheduled tasks (register with k.scheduler.Command(...).Cron("...").Register() etc.)
func (k *Kernel) Schedule() {
	logger.Debug(context.Background(), "scheduled tasks initialized (none registered by default)")
}

// Start starts the scheduler
func (k *Kernel) Start(ctx context.Context) error {
	logger.Info(ctx, "starting scheduler")
	k.Schedule()
	return k.scheduler.Start(ctx)
}

// Stop stops the scheduler
func (k *Kernel) Stop() {
	logger.Info(context.Background(), "stopping scheduler")
	k.scheduler.Stop()
}
