package commands

import (
	"context"

	"go-admin-scaffold/internal/database/migrations"
	"go-admin-scaffold/pkg/console"
	"go-admin-scaffold/pkg/database"
)

type MigrateCommand struct {
	*console.BaseCommand
}

func NewMigrateCommand() *MigrateCommand {
	return &MigrateCommand{
		BaseCommand: console.NewCommand("migrate", "Run database migrations"),
	}
}

func (c *MigrateCommand) Configure(config *console.CommandConfig) {
	config.Name = "migrate"
	config.Description = "Run database migrations"
}

func (c *MigrateCommand) Handle(ctx context.Context) error {
	db, err := database.FromContext(ctx)
	if err != nil {
		return err
	}
	migrator := migrations.InitMigrations(db)
	return migrator.RunPending()
}
