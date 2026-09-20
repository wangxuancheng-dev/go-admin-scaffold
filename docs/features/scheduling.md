# 任务调度

## 概述

任务调度系统依赖 Redis 进行分布式锁管理。确保在 `configs/config.yaml` 中正确配置 Redis：

```yaml
redis:
  host: "localhost"
  port: "6379"
  password: ""
  db: 0
```

异步队列（Asynq）使用同一 Redis 实例时：**`queue.connection.redis`** 可写完整 URL；否则用 **`redis.host` / `port` / `password`**，库号由 **`queue.connection.db`** 指定（可省略，则同 **`redis.db`**），见 [配置说明](../getting-started/configuration.md) 与 [队列系统](queue.md)。

## 创建任务

1. 创建命令：

```go
// internal/commands/backup_command.go
package commands

import (
    "context"
    "go-admin-scaffold/pkg/console"
)

type BackupCommand struct {
    *console.BaseCommand
}

func NewBackupCommand() *BackupCommand {
    return &BackupCommand{
        BaseCommand: console.NewCommand("backup:run", "Run database backup"),
    }
}

func (c *BackupCommand) Handle(ctx context.Context) error {
    // 实现备份逻辑
    return nil
}
```

2. 注册命令：

```go
// cmd/server/main.go
manager.Register(commands.NewBackupCommand())
```

## 调度任务

在 `internal/schedule/kernel.go` 中定义调度计划：

```go
func (k *Kernel) Schedule() {
    // 每天凌晨2点执行备份
    k.scheduler.Command("backup:run").Daily().At("02:00").Unique().Register()

    // 每30分钟清理缓存
    k.scheduler.Command("cache:clear").EveryThirtyMinutes().Register()

    // 每周日凌晨执行数据统计
    k.scheduler.Command("stats:generate").Weekly().At("00:00").Unique().Register()
}
```

## 调度选项

### 时间间隔

```go
// 每分钟执行
scheduler.Command("task").EveryMinute().Register()

// 每5分钟执行
scheduler.Command("task").EveryFiveMinutes().Register()

// 每小时执行
scheduler.Command("task").Hourly().Register()

// 每天执行
scheduler.Command("task").Daily().Register()

// 每周执行
scheduler.Command("task").Weekly().Register()

// 每月执行
scheduler.Command("task").Monthly().Register()

// 自定义Cron表达式
scheduler.Command("task").Cron("*/15 * * * *").Register()
```

### 指定时间

```go
// 每天特定时间执行
scheduler.Command("task").Daily().At("15:00").Register()

// 工作日执行
scheduler.Command("task").Weekdays().At("13:00").Register()

// 周末执行
scheduler.Command("task").Weekends().At("13:00").Register()
```

## 分布式环境

### 唯一任务

> **说明**：此处的「唯一」指 **定时调度** 在多机部署时由分布式锁保证 **Cron 任务只在一台机器执行**，与 [队列唯一任务（按业务键入队去重）](queue.md) 不同。

使用 `.Unique()` 确保任务只在一台服务器上执行：

```go
// 在多台服务器中只有一台会执行此任务
scheduler.Command("report:generate").Daily().At("00:00").Unique().Register()
```

### 普通任务

不使用 `.Unique()` 的任务会在所有服务器上执行：

```go
// 所有服务器都会执行此任务
scheduler.Command("cache:clear").Hourly().Register()
```

## 任务锁定

系统使用 Redis 实现分布式锁：

- 锁超时时间默认为 30 分钟
- 任务完成后自动释放锁
- 如果任务执行时间超过锁超时时间，其他服务器可能会重复执行

## 错误处理

任务执行错误会被记录到日志中：

```go
func (c *BackupCommand) Handle(ctx context.Context) error {
    if err := performBackup(); err != nil {
        // 错误会被记录到日志
        return fmt.Errorf("backup failed: %v", err)
    }
    return nil
}
```

## 监控和日志

- 任务执行状态记录在日志文件中
- 可以通过日志查看任务执行情况
- 建议配置日志聚合系统进行监控

## 最佳实践

1. 合理使用 Unique：
   - 数据备份、报表生成等任务应该设置为 Unique
   - 缓存清理、日志轮转等任务可以在所有服务器上执行

2. 设置合适的执行时间：
   - 避免在业务高峰期执行重任务
   - 分散不同任务的执行时间

3. 错误处理：
   - 实现合适的重试机制
   - 记录详细的错误信息
   - 配置监控和告警

4. 任务设计：
   - 保持任务的原子性
   - 实现幂等性，防止重复执行
   - 合理估计执行时间，设置适当的锁超时时间 