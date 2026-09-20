package main

import (
	"context"
	"log"

	"go-admin-scaffold/cmd/tools/commands"
	"go-admin-scaffold/internal/bootstrap"
	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/pkg/console"
	"go-admin-scaffold/pkg/database"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	db, err := bootstrap.SetupDatabase(cfg)
	if err != nil {
		log.Fatal(err)
	}

	manager := console.NewManager()
	manager.Register(commands.NewMigrateCommand())
	manager.Register(commands.NewSeedCommand())
	manager.Register(commands.NewMakeMigrationCommand())

	ctx := database.WithContext(context.Background(), db)
	if err := manager.RunFromArgsWithContext(ctx); err != nil {
		log.Fatal(err)
	}
}
