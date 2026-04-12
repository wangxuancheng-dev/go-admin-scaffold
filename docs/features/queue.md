# 队列系统

本文档详细说明了 Go Admin Scaffold 的队列系统功能和使用方法。

## 系统概述

队列系统支持多种驱动（Redis、数据库），用于处理异步任务，如邮件发送、文件处理、数据同步等。

### 主要特性

- 多驱动支持（Redis、数据库）
- 任务优先级管理
- 失败重试机制
- 任务超时控制
- **唯一队列（Unique Job）**：同一队列 + 相同业务键在排队/处理未结束前只接受一次入队（见下文）
- 任务状态监控
- 命令行工具支持

## 唯一队列（Unique Job）

与 [定时任务里的 `.Unique()`](scheduling.md#分布式环境) **不是同一概念**：那里是多机调度时只有一个实例跑 **Cron**；这里是 **异步队列** 里按业务键（如订单号）去重，类似 Laravel `ShouldBeUnique`。

### 结构体任务：设置 `BaseJob.UniqueKey`

项目内示例见 `internal/core/jobs/example_job.go` 中的 `ProcessOrderJob` / `NewProcessOrderJob`：

```go
import (
    "errors"
    "app/internal/core/jobs"
    "app/pkg/queue"
)

job := jobs.NewProcessOrderJob("20250412001", "recalculate_total")
if err := queueService.Push(ctx, job); err != nil {
    if errors.Is(err, queue.ErrDuplicateJob) {
        // 同一订单任务已在队列或执行中，可按 409 等语义返回
        return err
    }
    return err
}
```

也可用 `queue.NewBaseJob` 的 `options` 传入 `"unique_key"`（字符串）。

### `PushRaw` 与 options

```go
payload := []byte(`{"queue":"default","message":"ping"}`)
err := queueService.PushRaw(ctx, "default", payload, map[string]interface{}{
    "unique_key": "raw:order:20250412001",
})
// 若 payload 为 JSON 对象且不含 unique_key，驱动会把 options 里的键合并进 payload，便于完成后释放 Redis 锁 / DB 唯一约束
```

### 配置：Redis 锁 TTL

在应用配置中（秒）：

```yaml
queue:
  unique_ttl: 86400   # 可选；0 或未设置则使用驱动默认（约 24h 或与任务 timeout 相关）
```

### 返回值

重复入队时返回 `queue.ErrDuplicateJob`，请使用 `errors.Is(err, queue.ErrDuplicateJob)` 判断。

## 配置说明

### 1. 基础配置

```yaml
queue:
  default: "redis"         # 默认队列驱动
  connections:
    redis:
      driver: "redis"      # Redis驱动
      queue: "default"     # 队列名称
      retry_after: 90      # 重试等待时间(秒)
      timeout: 60          # 任务超时时间(秒)
    database:
      driver: "database"   # 数据库驱动
      table: "jobs"        # 任务表名
      queue: "default"     # 队列名称
      retry_after: 90      # 重试等待时间(秒)
      timeout: 60          # 任务超时时间(秒)
```

### 2. 环境变量

```bash
QUEUE_CONNECTION=redis     # 默认队列驱动
QUEUE_RETRY_AFTER=90      # 重试等待时间
QUEUE_TIMEOUT=60          # 任务超时时间
```

## 使用方法

### 1. 创建任务

```go
import "github.com/1768177868/go-admin-scaffold/pkg/queue"

// 创建任务
job := queue.NewJob("send_email", map[string]interface{}{
    "to": "user@example.com",
    "subject": "Welcome",
    "body": "Welcome to our platform",
})

// 设置任务选项
job.OnQueue("high")           // 设置队列
job.Delay(5 * time.Minute)    // 延迟执行
job.Timeout(30 * time.Second) // 设置超时
job.Retries(3)               // 设置重试次数

// 分发任务
err := queue.Dispatch(job)
```

### 2. 处理任务

```go
// 定义任务处理器
type EmailJob struct {
    To      string
    Subject string
    Body    string
}

func (j *EmailJob) Handle() error {
    // 处理发送邮件逻辑
    return nil
}

// 注册任务处理器
queue.Register("send_email", &EmailJob{})

// 启动队列处理
queue.Start()
```

### 3. 任务状态

```go
// 获取任务状态
status, err := queue.GetJobStatus(jobID)

// 检查任务是否完成
if status.IsCompleted() {
    // 处理完成逻辑
}

// 获取任务结果
result, err := queue.GetJobResult(jobID)
```

## 命令行工具

### 1. 启动队列服务

```bash
# 使用默认配置启动
./queue-cmd.exe -start

# 指定配置文件启动
./queue-cmd.exe -config=configs/production.yaml -start

# 直接运行worker
./worker.exe
```

### 2. 查看队列状态

```bash
# 查看所有队列状态
./queue-status.exe -all

# 查看特定队列状态
./queue-status.exe -queue=default

# 查看特定驱动的队列状态
./queue-status.exe -queue=high -driver=database
```

### 3. 管理任务

```bash
# 重试失败的任务
./queue-cmd.exe -retry=job_id

# 删除任务
./queue-cmd.exe -delete=job_id

# 清空队列
./queue-cmd.exe -flush=queue_name
```

## 开发环境

### 1. Windows 环境

```bash
# 启动队列服务
./queue-cmd.exe -start

# 后台运行
Start-Process -NoNewWindow -FilePath "./worker.exe"

# 检查状态
./queue-status.exe -all
```

### 2. Mac 环境

```bash
# 启动队列服务
./queue-cmd -start

# 后台运行
nohup ./worker > worker.log 2>&1 &

# 检查状态
./queue-status -all
```

## 生产环境

### 1. 进程管理

#### Linux (systemd)
```ini
[Unit]
Description=Go Admin Queue Worker
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/path/to/app
ExecStart=/path/to/app/worker
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

#### Windows (NSSM)
```powershell
# 安装服务
nssm install GoAdminQueue "C:\path\to\worker.exe"
nssm set GoAdminQueue AppDirectory "C:\path\to\app"
nssm set GoAdminQueue DisplayName "Go Admin Queue Worker"
nssm set GoAdminQueue Description "Go Admin Queue Worker Service"
nssm set GoAdminQueue Start SERVICE_AUTO_START
nssm start GoAdminQueue
```

### 2. 最佳实践

1. 配置管理
   - 使用单独的配置文件
   - 敏感信息使用环境变量
   - 定期检查配置有效性

2. 监控告警
   - 监控队列长度
   - 监控处理延迟
   - 监控失败任务
   - 设置告警阈值

3. 性能优化
   - 合理设置并发数
   - 优化任务处理逻辑
   - 使用适当的队列驱动
   - 定期清理过期任务

4. 高可用
   - 多worker部署
   - 任务重试机制
   - 故障自动恢复
   - 数据备份策略

## 故障排除

### 1. 任务未处理

检查：
- 队列服务是否运行
- 任务是否正确分发
- 处理器是否正确注册
- 日志中是否有错误

### 2. 任务积压

解决：
- 增加worker数量
- 优化任务处理逻辑
- 检查系统资源使用
- 考虑任务优先级

### 3. 内存使用过高

解决：
- 检查任务数据大小
- 优化任务处理逻辑
- 调整worker数量
- 监控内存使用

### 4. 连接问题

解决：
- 检查Redis/数据库连接
- 验证网络连接
- 检查认证信息
- 查看连接日志

## 维护命令

### 1. 日常维护

```bash
# 查看队列状态
./queue-status.exe -all

# 清理过期任务
./queue-cmd.exe -cleanup

# 重置失败任务
./queue-cmd.exe -reset-failed

# 导出队列统计
./queue-cmd.exe -export-stats
```

### 2. 故障恢复

```bash
# 重启队列服务
./queue-cmd.exe -restart

# 重置特定队列
./queue-cmd.exe -reset=queue_name

# 恢复失败任务
./queue-cmd.exe -recover-failed

# 检查队列健康
./queue-cmd.exe -health-check
```

## 相关文档

- [配置说明](../getting-started/configuration.md)
- [开发环境配置](../advanced/development.md)
- [部署指南](../deployment/README.md)
- [API 文档](../api/README.md) 