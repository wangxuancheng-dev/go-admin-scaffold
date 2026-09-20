package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-admin-scaffold/pkg/console"
)

type MakeMigrationCommand struct {
	*console.BaseCommand
}

func NewMakeMigrationCommand() *MakeMigrationCommand {
	cmd := &MakeMigrationCommand{
		BaseCommand: console.NewCommand("make:migration", "Create a new migration file"),
	}
	return cmd
}

func (c *MakeMigrationCommand) Configure(config *console.CommandConfig) {
	config.Name = "make:migration"
	config.Description = "Create a new migration file"
	config.Usage = "make:migration [name] [--create=table_name] [--table=table_name]"
	config.Arguments = []console.Argument{
		{
			Name:        "name",
			Description: "The name of the migration (e.g., create_users_table)",
			Required:    true,
		},
	}
	config.Options = []console.Option{
		{
			Name:        "create",
			Description: "Create a new table",
		},
		{
			Name:        "table",
			Description: "Modify an existing table",
		},
	}
}

func (c *MakeMigrationCommand) Handle(ctx context.Context) error {
	args := ctx.Value("args").([]string)
	if len(args) < 2 {
		return fmt.Errorf("migration name required")
	}

	name := args[1]

	// Parse options
	var createTable, modifyTable string
	for i := 2; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--create=") {
			createTable = strings.TrimPrefix(arg, "--create=")
		} else if strings.HasPrefix(arg, "--table=") {
			modifyTable = strings.TrimPrefix(arg, "--table=")
		}
	}

	// Generate timestamp in Laravel format: 2024_03_15_123456
	timestamp := time.Now().Format("2006_01_02_150405")
	filename := fmt.Sprintf("%s_%s.go", timestamp, strings.ToLower(name))

	// Create migrations directory if it doesn't exist
	migrationsDir := "internal/database/migrations"
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %v", err)
	}

	// Create migration file
	filepath := filepath.Join(migrationsDir, filename)

	var content string
	if createTable != "" {
		content = generateCreateTableMigration(name, createTable)
	} else if modifyTable != "" {
		content = generateModifyTableMigration(name, modifyTable)
	} else {
		content = generateGenericMigration(name)
	}

	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create migration file: %v", err)
	}

	fmt.Printf("Created Migration: %s\n", filename)
	return nil
}

func generateCreateTableMigration(name, tableName string) string {
	return fmt.Sprintf(`package migrations

import (
	"time"
	"gorm.io/gorm"
)

func init() {
	Register("%s", &MigrationDefinition{
		Up: func(tx *gorm.DB) error {
			// Create %s table
			type %s struct {
				ID        uint           `+"`gorm:\"primarykey\"`"+`
				// Add your columns here
				CreatedAt time.Time      `+"`gorm:\"type:timestamp\"`"+`
				UpdatedAt time.Time      `+"`gorm:\"type:timestamp\"`"+`
				DeletedAt gorm.DeletedAt `+"`gorm:\"index;type:timestamp\"`"+`
			}

			return tx.Table("%s").AutoMigrate(&%s{})
		},
		Down: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable("%s")
		},
	})
}
`, name, tableName, toStructName(tableName), tableName, toStructName(tableName), tableName)
}

func generateModifyTableMigration(name, tableName string) string {
	return fmt.Sprintf(`package migrations

import (
	"gorm.io/gorm"
)

func init() {
	Register("%s", &MigrationDefinition{
		Up: func(tx *gorm.DB) error {
			// Modify %s table
			// Example: Add a new column
			// if err := tx.Exec("ALTER TABLE %s ADD COLUMN new_column VARCHAR(255)").Error; err != nil {
			//     return err
			// }
			
			// Example: Modify existing column
			// if err := tx.Exec("ALTER TABLE %s MODIFY COLUMN existing_column VARCHAR(255) NOT NULL").Error; err != nil {
			//     return err
			// }
			
			return nil
		},
		Down: func(tx *gorm.DB) error {
			// Reverse the changes made in Up method
			// Example: Drop the added column
			// if err := tx.Exec("ALTER TABLE %s DROP COLUMN new_column").Error; err != nil {
			//     return err
			// }
			
			return nil
		},
	})
}
`, name, tableName, tableName, tableName, tableName)
}

func generateGenericMigration(name string) string {
	return fmt.Sprintf(`package migrations

import (
	"gorm.io/gorm"
)

func init() {
	Register("%s", &MigrationDefinition{
		Up: func(tx *gorm.DB) error {
			// TODO: Implement your migration logic here
			// Example: Create tables, add columns, etc.
			return nil
		},
		Down: func(tx *gorm.DB) error {
			// TODO: Implement rollback logic here
			// Example: Drop tables, remove columns, etc.
			return nil
		},
	})
}
`, name)
}

func toStructName(name string) string {
	// Convert snake_case to PascalCase (ASCII identifiers only).
	words := strings.Split(name, "_")
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
	}
	return strings.Join(words, "")
}
