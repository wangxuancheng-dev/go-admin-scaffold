package commands

import (
	"context"
	"fmt"

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
		BaseCommand: console.NewCommand("seed", "Database seeding commands"),
	}
	return cmd
}

func (c *SeedCommand) Configure(config *console.CommandConfig) {
	config.Name = "seed"
	config.Description = "Database seeding commands"
	config.Usage = "seed [action] [seeder?]"
	config.Arguments = []console.Argument{
		{
			Name:        "action",
			Description: "Action to perform (run, reset, or status)",
			Required:    true,
		},
		{
			Name:        "seeder",
			Description: "Optional seeder name(s) to run",
			Required:    false,
		},
	}
}

func (c *SeedCommand) Handle(ctx context.Context) error {
	args := ctx.Value("args").([]string)
	if len(args) < 2 {
		return fmt.Errorf("action required: run, reset, or status")
	}

	db, err := database.FromContext(ctx)
	if err != nil {
		return err
	}
	seederManager := seeder.NewSeederManager(db)
	seeders.SetGlobalManager(seederManager)

	action := args[1]
	switch action {
	case "run":
		// If specific seeders are provided, run only those
		if len(args) > 2 {
			return seederManager.Run(args[2:]...)
		}
		// Otherwise run all seeders
		return seederManager.Run()

	case "reset":
		return seederManager.Reset()

	case "status":
		status, err := seederManager.Status()
		if err != nil {
			return err
		}
		printSeederStatus(status)
		return nil

	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func printSeederStatus(status []map[string]interface{}) {
	fmt.Println("\nSeeder Status:")
	fmt.Println("------------------------------------------")
	fmt.Printf("%-30s %-10s %-30s\n", "Seeder", "Status", "Description")
	fmt.Println("------------------------------------------")

	for _, s := range status {
		name := s["name"].(string)
		status := s["status"].(string)
		description := s["description"].(string)
		fmt.Printf("%-30s %-10s %-30s\n", name, status, description)
	}
	fmt.Println("------------------------------------------")
}
