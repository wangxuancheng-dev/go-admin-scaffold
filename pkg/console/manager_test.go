package console_test

import (
	"context"
	"os"
	"testing"

	"go-admin-scaffold/pkg/console"

	"github.com/stretchr/testify/require"
)

type stubCmd struct {
	*console.BaseCommand
	called bool
}

func (s *stubCmd) Configure(cfg *console.CommandConfig) {
	cfg.Name = "ping"
	cfg.Description = "ping"
}

func (s *stubCmd) Handle(ctx context.Context) error {
	s.called = true
	return nil
}

func TestManager_RunFromArgsWithContext(t *testing.T) {
	m := console.NewManager()
	cmd := &stubCmd{BaseCommand: console.NewCommand("ping", "ping")}
	m.Register(cmd)

	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"prog", "ping"}

	require.NoError(t, m.RunFromArgsWithContext(context.Background()))
	require.True(t, cmd.called)
}

func TestManager_unknownCommand(t *testing.T) {
	m := console.NewManager()
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"prog", "nope"}
	require.Error(t, m.RunFromArgs())
}
