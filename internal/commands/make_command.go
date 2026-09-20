package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-admin-scaffold/pkg/console"
)

type MakeCommand struct {
	*console.BaseCommand
}

func NewMakeCommand() *MakeCommand {
	cmd := &MakeCommand{
		BaseCommand: console.NewCommand("make", "Generate code files"),
	}

	cmd.AddArgument("type", "Type of file to create (controller/model/service)")
	cmd.AddArgument("name", "Name of the file to create")

	return cmd
}

func (c *MakeCommand) Handle(ctx context.Context) error {
	fileType := c.GetArgument("type")
	name := c.GetArgument("name")

	switch strings.ToLower(fileType) {
	case "controller":
		return c.makeController(name)
	case "model":
		return c.makeModel(name)
	case "service":
		return c.makeService(name)
	default:
		return fmt.Errorf("unknown type: %s", fileType)
	}
}

func (c *MakeCommand) makeController(name string) error {
	template := `package controllers

type %sController struct {}

func New%sController() *%sController {
	return &%sController{}
}
`
	return c.createFile("internal/api/admin/controllers", name+"_controller.go", template, name, name, name, name)
}

func (c *MakeCommand) makeModel(name string) error {
	template := `package models

type %s struct {
	ID        uint      ` + "`json:\"id\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\"`" + `
}
`
	return c.createFile("internal/core/models", name+".go", template, name)
}

func (c *MakeCommand) makeService(name string) error {
	template := `package services

type %sService struct {}

func New%sService() *%sService {
	return &%sService{}
}
`
	return c.createFile("internal/core/services", name+"_service.go", template, name, name, name, name)
}

func (c *MakeCommand) createFile(dir, filename, template string, args ...interface{}) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, template, args...)
	return err
}
