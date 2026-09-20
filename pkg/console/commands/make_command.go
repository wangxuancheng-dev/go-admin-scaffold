package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"go-admin-scaffold/pkg/console"
)

type MakeCommand struct {
	console.BaseCommand
}

func NewMakeCommand() *MakeCommand {
	return &MakeCommand{}
}

func (c *MakeCommand) Configure(config *console.CommandConfig) {
	config.Name = "make:command"
	config.Description = "Create a new Artisan command"
	config.Usage = "make:command [command-name]"
	config.Arguments = []console.Argument{
		{
			Name:        "name",
			Description: "The name of the command",
			Required:    true,
		},
	}
	c.BaseCommand.Configure(config)
}

func (c *MakeCommand) Handle(ctx context.Context) error {
	name := c.GetArgument("name")
	if name == "" {
		return fmt.Errorf("command name is required")
	}

	// Create commands directory if it doesn't exist
	cmdDir := "internal/commands"
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		return fmt.Errorf("failed to create commands directory: %v", err)
	}

	// Convert command name to file name
	fileName := strings.ToLower(name)
	if !strings.HasSuffix(fileName, "_command") {
		fileName += "_command"
	}
	fileName = filepath.Join(cmdDir, fileName+".go")

	// Check if file already exists
	if _, err := os.Stat(fileName); err == nil {
		return fmt.Errorf("command file already exists: %s", fileName)
	}

	// Create command file from template
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create command file: %v", err)
	}
	defer file.Close()

	// Parse command name
	cmdName := strings.TrimSuffix(name, "Command")
	cmdName = strings.TrimSuffix(cmdName, "_command")
	cmdName = strings.TrimSuffix(cmdName, "-command")

	// Generate command file content
	tmpl := template.Must(template.New("command").Parse(commandTemplate))
	err = tmpl.Execute(file, map[string]interface{}{
		"Name":      cmdName,
		"ClassName": strings.Title(cmdName) + "Command",
	})
	if err != nil {
		return fmt.Errorf("failed to generate command file: %v", err)
	}

	c.Success("Command created successfully: %s", fileName)
	return nil
}

const commandTemplate = `package commands

import (
	"context"

	"go-admin-scaffold/pkg/console"
)

type {{.ClassName}} struct {
	console.BaseCommand
}

func New{{.ClassName}}() *{{.ClassName}} {
	return &{{.ClassName}}{}
}

func (c *{{.ClassName}}) Configure(config *console.CommandConfig) {
	config.Name = "{{.Name}}"
	config.Description = "Description of {{.Name}} command"
	config.Usage = "{{.Name}} [arguments]"
	
	// Add your command arguments and options here
	config.Arguments = []console.Argument{
		// {
		//     Name:        "argument",
		//     Description: "Argument description",
		//     Required:    true,
		// },
	}
	
	config.Options = []console.Option{
		// {
		//     Name:        "option",
		//     Shortcut:    "o",
		//     Description: "Option description",
		//     Required:    false,
		// },
	}
	
	c.BaseCommand.Configure(config)
}

func (c *{{.ClassName}}) Handle(ctx context.Context) error {
	// Implement your command logic here
	c.Info("Running {{.Name}} command")
	return nil
}
`
