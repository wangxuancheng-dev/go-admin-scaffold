package commands

import (
	"context"
	"flag"

	"go-admin-scaffold/internal/database/seeder"
	"go-admin-scaffold/internal/database/seeders"
	"go-admin-scaffold/pkg/console"
	"go-admin-scaffold/pkg/database"
)

type SeedCommand struct {
	*console.BaseCommand
}

func NewSeedCommand() *SeedCommand {
	cmd := &SeedCommand{
		BaseCommand: console.NewCommand("seed", "Manage database seeding"),
	}
	return cmd
}

func (c *SeedCommand) Configure(config *console.CommandConfig) {
	config.Name = "seed"
	config.Description = "Manage database seeding"
}

func (c *SeedCommand) Handle(ctx context.Context) error {
	db, err := database.FromContext(ctx)
	if err != nil {
		return err
	}
	seederManager := seeder.NewSeederManager(db)

	// Set the global manager for seeders to register themselves
	seeders.SetGlobalManager(seederManager)

	// Parse command arguments
	args := flag.Args()
	if len(args) < 2 {
		return c.showUsage()
	}

	switch args[1] {
	case "run":
		if len(args) > 2 {
			// Run specific seeders
			return seederManager.Run(args[2:]...)
		}
		// Run all seeders
		return seederManager.Run()
	case "status":
		return c.showStatus(seederManager)
	case "reset":
		return seederManager.Reset()
	default:
		return c.showUsage()
	}
}

func (c *SeedCommand) showUsage() error {
	println("Usage:")
	println("  seed run [names...]  - Run seeders")
	println("  seed status          - Show seeder status")
	println("  seed reset           - Reset all seeded data")
	return nil
}

func (c *SeedCommand) showStatus(manager *seeder.SeederManager) error {
	status, err := manager.Status()
	if err != nil {
		return err
	}

	println()
	println("Seeder Status:")
	println("------------------------------------------")
	println("Seeder                         Status     Description")
	println("------------------------------------------")

	for _, s := range status {
		println(s["name"].(string), s["status"].(string), s["description"].(string))
	}
	println("------------------------------------------")

	return nil
}
