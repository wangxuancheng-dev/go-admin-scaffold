package console

import (
	"context"
	"fmt"
	"os"
)

type ctxKey int

const argsCtxKey ctxKey = 1

// Manager manages console commands
type Manager struct {
	commands map[string]Command
}

// NewManager creates a new command manager
func NewManager() *Manager {
	return &Manager{
		commands: make(map[string]Command),
	}
}

// Register registers a command
func (m *Manager) Register(cmd Command) {
	config := &CommandConfig{}
	cmd.Configure(config)
	m.commands[config.Name] = cmd
}

// FindCommand finds a command by name
func (m *Manager) FindCommand(name string) Command {
	return m.commands[name]
}

// RunFromArgs runs a command from command line arguments
func (m *Manager) RunFromArgs() error {
	return m.RunFromArgsWithContext(context.Background())
}

// RunFromArgsWithContext runs a command using base context (attach DB via database.WithContext).
func (m *Manager) RunFromArgsWithContext(base context.Context) error {
	if base == nil {
		base = context.Background()
	}
	args := os.Args[1:]
	if len(args) == 0 {
		return m.showAvailableCommands()
	}

	cmdName := args[0]
	cmd := m.FindCommand(cmdName)
	if cmd == nil {
		return fmt.Errorf("command not found: %s", cmdName)
	}

	ctx := context.WithValue(base, argsCtxKey, args)
	return cmd.Handle(ctx)
}

// showAvailableCommands shows all available commands
func (m *Manager) showAvailableCommands() error {
	fmt.Println("Available commands:")
	for name, cmd := range m.commands {
		fmt.Printf("  %s\t%s\n", name, cmd.GetDescription())
	}
	return nil
}
