# 命令行工具

脚手架提供 `cmd/artisan`（业务命令）与 `cmd/tools`（迁移生成等工具）。

## Artisan（`cmd/artisan`）

已注册命令（以 `cmd/artisan/main.go` 为准）：

| 命令 | 说明 |
|------|------|
| `make` | 生成 handler / model / service |
| `migrate` | 运行 / 回滚迁移 |
| `seed` | 运行 seeder |
| `schedule:run` | 执行一次调度内核 |

```bash
go run ./cmd/artisan
go run ./cmd/artisan help
go run ./cmd/artisan make handler Product
go run ./cmd/artisan make model Product
go run ./cmd/artisan make service Product
go run ./cmd/artisan migrate run
go run ./cmd/artisan seed run
go run ./cmd/artisan schedule:run
```

`make handler` 写入 `internal/api/admin/v1/`（`controller` 别名仍可用，路径相同）。

## Tools（`cmd/tools`）

迁移文件生成等工具在 tools 入口：

```bash
go run ./cmd/tools make:migration create_xxx_table
go run ./cmd/tools migrate run
go run ./cmd/tools seed run
```

Swagger 文档用 swag（CI 也会校验 drift）：

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g ./cmd/server/main.go -o ./docs
```

## 自定义命令

```go
// internal/commands/ping_command.go
package commands

import (
	"context"
	"time"

	"go-admin-scaffold/pkg/console"
)

type PingCommand struct {
	*console.BaseCommand
}

func NewPingCommand() *PingCommand {
	return &PingCommand{
		BaseCommand: console.NewCommand("app:ping", "Print a timestamp"),
	}
}

func (c *PingCommand) Handle(ctx context.Context) error {
	c.Line("pong at %s", time.Now().Format(time.RFC3339))
	return nil
}
```

在 `cmd/artisan/main.go` 中 `manager.Register(commands.NewPingCommand())`。

数据库命令通过 `database.WithContext` / `FromContext` 取 `*gorm.DB`，见 [architecture.md](architecture.md)。

## 相关

- [测试](testing.md)
- [迁移](../database/migrations.md)
- [调度](../features/scheduling.md)
