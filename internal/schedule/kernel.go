package schedule

import (
	"context"
	"log"
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
	log.Println("Scheduled tasks initialized (none registered by default)")
}

// Start starts the scheduler
func (k *Kernel) Start(ctx context.Context) error {
	log.Println("Starting scheduler...")
	k.Schedule()
	return k.scheduler.Start(ctx)
}

// Stop stops the scheduler
func (k *Kernel) Stop() {
	log.Println("Stopping scheduler...")
	k.scheduler.Stop()
}
