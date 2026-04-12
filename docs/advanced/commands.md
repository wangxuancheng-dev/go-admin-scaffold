# 命令行工具

本框架提供了强大的命令行工具系统，支持自定义命令、代码生成和数据库操作等功能。

## 基本使用

### 1. 可用命令

```bash
# 查看所有可用命令
go run cmd/artisan/main.go

# 查看命令帮助
go run cmd/artisan/main.go help [command]
```

### 2. 内置命令

```bash
# 创建迁移文件
go run cmd/artisan/main.go make:migration create_users_table

# 运行迁移
go run cmd/artisan/main.go migrate

# 创建模型
go run cmd/artisan/main.go make:model User

# 创建控制器
go run cmd/artisan/main.go make:controller UserController

# 创建服务
go run cmd/artisan/main.go make:service UserService

# 运行定时任务
go run cmd/artisan/main.go schedule:run

# 生成 API 文档
go run cmd/artisan/main.go docs:generate

# 缓存配置
go run cmd/artisan/main.go config:cache
```

## 创建自定义命令

### 1. 基本命令结构

```go
// internal/commands/ping_command.go
package commands

import (
    "context"
    "time"

    "app/pkg/console"
)

type PingCommand struct {
    console.BaseCommand
}

func NewPingCommand() *PingCommand {
    return &PingCommand{}
}

func (c *PingCommand) Configure(config *console.CommandConfig) {
    config.Name = "app:ping"
    config.Description = "Print a timestamp"
    config.Usage = "app:ping"
    c.BaseCommand.Configure(config)
}

func (c *PingCommand) Handle(ctx context.Context) error {
    c.Line("pong at %s", time.Now().Format(time.RFC3339))
    return nil
}
```

### 2. 带参数的命令

```go
// internal/commands/greet_command.go
type GreetCommand struct {
    *console.BaseCommand
    name string
}

func NewGreetCommand() *GreetCommand {
    cmd := &GreetCommand{
        BaseCommand: console.NewCommand(
            "greet",
            "Greet a user",
        ),
    }
    
    cmd.AddArgument("name", "Name of the user")
    cmd.AddOption("uppercase", "u", "Convert to uppercase")
    
    return cmd
}

func (c *GreetCommand) Handle(ctx context.Context) error {
    name := c.GetArgument("name")
    if c.GetOption("uppercase") {
        name = strings.ToUpper(name)
    }
    
    console.Info("Hello, %s!", name)
    return nil
}
```

### 3. 进度显示

```go
func (c *ImportCommand) Handle(ctx context.Context) error {
    bar := console.NewProgressBar(100)
    
    for i := 0; i < 100; i++ {
        // 处理逻辑
        time.Sleep(50 * time.Millisecond)
        bar.Advance()
    }
    
    bar.Finish()
    return nil
}
```

## 注册命令

### 1. 在主程序中注册

```go
// cmd/artisan/main.go
func main() {
    manager := console.NewManager()

    // 注册内置命令
    manager.Register(commands.NewMigrateCommand())
    manager.Register(commands.NewSeedCommand())
    
    // 注册自定义命令
    manager.Register(commands.NewPingCommand())
    manager.Register(commands.NewGreetCommand())

    if err := manager.RunFromArgs(); err != nil {
        log.Fatal(err)
    }
}
```

### 2. 使用命令组

```go
// internal/commands/groups/database_group.go
type DatabaseCommandGroup struct {
    *console.CommandGroup
}

func NewDatabaseCommandGroup() *DatabaseCommandGroup {
    group := &DatabaseCommandGroup{
        CommandGroup: console.NewCommandGroup("db", "Database commands"),
    }
    
    group.Add(commands.NewMigrateCommand())
    group.Add(commands.NewSeedCommand())
    
    return group
}
```

## 代码生成器

### 1. 创建生成器命令

```go
// internal/commands/make_model_command.go
type MakeModelCommand struct {
    *console.BaseCommand
}

func (c *MakeModelCommand) Handle(ctx context.Context) error {
    name := c.GetArgument("name")
    
    // 生成模型文件
    template := `package models

type {{ .Name }} struct {
    ID        uint      ` + "`gorm:\"primarykey\"`" + `
    CreatedAt time.Time
    UpdatedAt time.Time
}

func ({{ .Name }}) TableName() string {
    return "{{ .Table }}"
}
`
    
    data := struct {
        Name  string
        Table string
    }{
        Name:  name,
        Table: strcase.ToSnake(name) + "s",
    }
    
    return c.GenerateFile(
        "internal/models/"+strcase.ToSnake(name)+".go",
        template,
        data,
    )
}
```

### 2. 生成控制器

```go
// internal/commands/make_controller_command.go
func (c *MakeControllerCommand) Handle(ctx context.Context) error {
    name := c.GetArgument("name")
    
    template := `package controllers

type {{ .Name }}Controller struct {
    service *services.{{ .Name }}Service
}

func New{{ .Name }}Controller(service *services.{{ .Name }}Service) *{{ .Name }}Controller {
    return &{{ .Name }}Controller{
        service: service,
    }
}

func (c *{{ .Name }}Controller) List(ctx *gin.Context) {
    // TODO: Implement list method
}

func (c *{{ .Name }}Controller) Create(ctx *gin.Context) {
    // TODO: Implement create method
}
`
    
    return c.GenerateFile(
        "internal/controllers/"+strcase.ToSnake(name)+"_controller.go",
        template,
        struct{ Name string }{Name: name},
    )
}
```

## 交互式命令

### 1. 用户输入

```go
func (c *SetupCommand) Handle(ctx context.Context) error {
    // 询问问题
    name := c.Ask("What is your name?")
    email := c.Ask("What is your email?")
    
    // 带验证的密码输入
    password := c.Secret("Enter password:", func(value string) error {
        if len(value) < 8 {
            return errors.New("password must be at least 8 characters")
        }
        return nil
    })
    
    // 选择选项
    role := c.Choice("Select role:", []string{
        "admin",
        "user",
        "guest",
    })
    
    // 确认
    if c.Confirm("Do you want to continue?", true) {
        // 执行设置
    }
    
    return nil
}
```

### 2. 表格输出

```go
func (c *ListUsersCommand) Handle(ctx context.Context) error {
    users, err := c.userService.GetAll()
    if err != nil {
        return err
    }
    
    table := console.NewTable([]string{"ID", "Name", "Email", "Role"})
    for _, user := range users {
        table.AddRow([]string{
            strconv.Itoa(int(user.ID)),
            user.Name,
            user.Email,
            user.Role,
        })
    }
    
    table.Render()
    return nil
}
```

## 最佳实践

1. 命令设计：
   - 使用清晰的命名约定
   - 提供有用的帮助信息
   - 支持合理的默认值

2. 错误处理：
   - 提供清晰的错误消息
   - 实现优雅的失败处理
   - 支持调试模式

3. 用户体验：
   - 提供进度反馈
   - 使用颜色突出重要信息
   - 支持交互式操作

4. 代码组织：
   - 按功能分组命令
   - 使用依赖注入
   - 保持代码可测试性

## 测试命令

```go
// tests/commands/ping_command_test.go
func TestPingCommand(t *testing.T) {
    cmd := commands.NewPingCommand()

    err := cmd.Handle(context.Background())

    assert.NoError(t, err)
}
``` 