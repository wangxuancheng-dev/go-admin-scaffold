package commands

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go-admin-scaffold/internal/schedule"
	"go-admin-scaffold/pkg/console"
)

type ScheduleRunCommand struct {
	console.BaseCommand
	kernel *schedule.Kernel
}

func NewScheduleRunCommand(kernel *schedule.Kernel) *ScheduleRunCommand {
	return &ScheduleRunCommand{
		kernel: kernel,
	}
}

func (c *ScheduleRunCommand) Configure(config *console.CommandConfig) {
	config.Name = "schedule:run"
	config.Description = "Run the scheduler"
	config.Usage = "schedule:run"
	c.BaseCommand.Configure(config)
}

func (c *ScheduleRunCommand) Handle(ctx context.Context) error {
	c.Info("Starting scheduler...")

	// Register scheduled tasks
	c.kernel.Schedule()

	// Start the scheduler
	if err := c.kernel.Start(ctx); err != nil {
		return err
	}

	c.Success("Scheduler started successfully")

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	c.Info("Shutting down scheduler...")
	c.kernel.Stop()
	c.Success("Scheduler stopped successfully")

	return nil
}
