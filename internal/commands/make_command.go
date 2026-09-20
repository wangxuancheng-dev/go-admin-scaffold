package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"go-admin-scaffold/pkg/console"
)

type MakeCommand struct {
	*console.BaseCommand
}

func NewMakeCommand() *MakeCommand {
	cmd := &MakeCommand{
		BaseCommand: console.NewCommand("make", "Generate code files"),
	}

	cmd.AddArgument("type", "Type of file to create (handler/model/service)")
	cmd.AddArgument("name", "Name of the file to create (e.g. Product)")

	return cmd
}

func (c *MakeCommand) Handle(ctx context.Context) error {
	fileType := c.GetArgument("type")
	name := c.GetArgument("name")
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name is required")
	}
	name = exportName(name)

	switch strings.ToLower(fileType) {
	case "handler", "controller":
		return c.makeHandler(name)
	case "model":
		return c.makeModel(name)
	case "service":
		return c.makeService(name)
	default:
		return fmt.Errorf("unknown type: %s (use handler, model, or service)", fileType)
	}
}

func exportName(name string) string {
	r := []rune(strings.TrimSpace(name))
	if len(r) == 0 {
		return name
	}
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func (c *MakeCommand) makeHandler(name string) error {
	lower := strings.ToLower(name)
	template := `package v1

import (
	"github.com/gin-gonic/gin"
)

// %sHandler handles %s endpoints.
type %sHandler struct{}

func New%sHandler() *%sHandler {
	return &%sHandler{}
}

// List is a stub — wire into routes.SetupRoutes after implementing.
func (h *%sHandler) List(c *gin.Context) {}
`
	return c.createFile("internal/api/admin/v1", lower+".go", template, name, lower, name, name, name, name, name)
}

func (c *MakeCommand) makeModel(name string) error {
	template := `package models

import "time"

type %s struct {
	ID        uint      ` + "`json:\"id\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\"`" + `
}
`
	return c.createFile("internal/core/models", strings.ToLower(name)+".go", template, name)
}

func (c *MakeCommand) makeService(name string) error {
	template := `package services

type %sService struct{}

func New%sService() *%sService {
	return &%sService{}
}
`
	return c.createFile("internal/core/services", strings.ToLower(name)+"_service.go", template, name, name, name, name)
}

func (c *MakeCommand) createFile(dir, filename, template string, args ...interface{}) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}
	content := fmt.Sprintf(template, args...)
	return os.WriteFile(path, []byte(content), 0o644)
}
