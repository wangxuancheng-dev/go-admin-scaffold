# 队列系统

本文档说明 Go Admin Scaffold 的异步队列：基于 **Redis** 与 **[Asynq](https://github.com/hibiken/asynq)**，用于耗时逻辑、通知、文件处理、数据同步等后台任务。

## 系统概述

- **入队**：`asynq.Client`（`pkg/queue` 的 `Manager.Push` / `Later` / `PushRaw`）。
- **消费**：`asynq.Server` + `ServeMux`，由 **`internal/core/jobs.RegisterAsynqHandlers`** 按任务类型分发并调用 `JobInterface.Handle()`。
- **存储与调度**：延迟、重试、归档等由 Asynq 在 Redis 中维护（非自建 Stream/ZSET）。

### 主要特性

- `driver` 为 `redis` 或 `asynq` 时均走 Asynq 实现
- 多队列与权重（`queue.queues` 的 `priority` → Asynq 队列权重）
- 并发度为各队列 `processes` 之和（`QueueService.Start` / `cmd/worker`）
- 任务级 `Timeout`、`MaxAttempts` 映射为 Asynq 的 `Timeout`、`MaxRetry`
- **唯一任务**：`BaseJob.UniqueKey` + Asynq `Unique(ttl)`；重复入队返回 `queue.ErrDuplicateJob`（`asynq.ErrDuplicateTask`）
- `Manager.Size` / `Clear` 通过 `asynq.Inspector`（统计为各状态任务总和）

## 任务类型与 `Handle`（job_type）

1. 在 `init` 中 **`queue.RegisterJobType("name", func() queue.JobInterface { return &YourJob{} })`**。
2. **`Manager.Push` / `Later`** 会为已注册的具体类型自动填入 **`BaseJob.JobType`**（与 Asynq 任务类型字符串一致）；也可手动设置 `JobType`。
3. 消费端必须在同一进程注册 **Asynq 处理器**：**`jobs.RegisterAsynqHandlers(mux)`**（见 `internal/core/jobs/asynq_handlers.go`）。业务侧需 **`import _ "app/internal/core/jobs"`** 或显式 import 含 `RegisterJobType` 的包，保证类型注册与 handler 一致。
4. 入队 payload 为 **整段任务 JSON**；Asynq 的 *task type* = `job_type` 字段。

### 与 `Pop` 的关系

当前驱动 **不提供** 拉取式 `Pop` / `Delete` / `Release`（返回 `queue.ErrPullNotSupported`）。消费请使用 **`cmd/worker`** 或应用内 **`QueueService.Start()`** 启动的 Asynq Server。

项目内参考：`internal/core/jobs/register.go`、`jobs.JobTypeExample` 等。

## 唯一队列（Unique Job）

与 [定时任务里的「唯一」](scheduling.md#分布式环境) **不同**：定时侧是多机 Cron 互斥；队列侧是按业务键（如订单号）在 **Asynq Unique 窗口内** 去重。

### 结构体任务

```go
import (
    "errors"
    "app/internal/core/jobs"
)

job := jobs.NewProcessOrderJob("20250412001", "recalculate_total")
if err := queueService.Push(ctx, job); err != nil {
    if errors.Is(err, queue.ErrDuplicateJob) {
        // 相同 Unique 约束下任务已存在
        return err
    }
    return err
}
```

也可用 `queue.NewBaseJob` 的 `options` 传入 `"unique_key"`（字符串）。

### `PushRaw`

```go
payload := []byte(`{"queue":"default","message":"ping"}`)
err := queueService.PushRaw(ctx, "default", payload, map[string]interface{}{
    "task_type":  "raw", // Asynq 任务类型；项目内 `raw` 注册为空操作 handler
    "unique_key": "raw:order:20250412001",
})
```

若 payload 为 JSON 对象且不含 `unique_key`，驱动会把 options 里的 `unique_key` 合并进 JSON，以便 Asynq `Unique` 与业务键一致。

### 配置：`unique_ttl`

业务 **`unique_key`** 在本项目中映射为稳定的 **`asynq.TaskID`**（由队列名 + 任务类型 + `unique_key` 派生），与是否跑 worker 无关；同一键在 Asynq 仍保留该任务 ID 期间再次入队会得到 **`ErrDuplicateJob`**（底层可能为 `ErrTaskIDConflict`）。

```yaml
queue:
  # 历史字段：此前配合 Asynq Unique(TTL) 使用；当前 unique_key 走 TaskID，此项可忽略或预留给后续扩展
  # unique_ttl: 86400
```

### 返回值

重复入队返回 **`queue.ErrDuplicateJob`**，请使用 **`errors.Is(err, queue.ErrDuplicateJob)`** 判断。

## 配置说明

与 `configs/config.yaml` 对齐要点：

```yaml
queue:
  driver: "redis"          # 或 asynq；均使用 Asynq
  queue: "default"         # Manager 默认队列名
  unique_ttl: 86400        # 可选，Unique 窗口（秒）
  connection:
    db: 1                  # 与顶层 redis 共用 host/port/password，仅换库；或写 redis: "redis://..." 覆盖整 URL
  worker:
    timeout: 60            # Asynq Server ShutdownTimeout（秒）
  queues:
    default:
      priority: 3
      processes: 1
```

历史字段 **`stream_maxlen` / `stream_trim_approx` / `stream_group`** 仍可出现于 YAML，**当前实现不读取**。

### 环境变量（节选）

与 `internal/config` 一致，例如：`QUEUE_DRIVER`、`QUEUE_NAME`、`REDIS_*` 等。队列 URL：`queue.connection.redis` 非空则用之；否则用 **`redis.host` / `port` / `password`** 与 **`queue.connection.db`**（若配置）或 **`redis.db`** 拼装（见 `QueueRedisConnectionURL`）。

## 使用方法

### 1. 定义任务并实现 `Handle`

嵌入 `queue.BaseJob`，实现 `queue.JobInterface`（含 `TaskType()`，默认来自嵌入字段 `JobType`）。

### 2. 注册类型与 Handler

- `RegisterJobType`：解码 JSON → 具体类型。
- `RegisterAsynqHandlers`：为每个 `job_type` 注册 `mux.HandleFunc`，内部 `queue.DecodeJobFromJSON` 后调用 `Handle()`。

新增任务类型时：**同时** 在 `register.go` 里 `RegisterJobType`，在 `asynq_handlers.go` 里 `register(JobTypeXxx)`（或改为循环注册表）。

### 3. 入队

```go
import (
    "context"
    "app/internal/core/jobs"
    "app/internal/core/services"
)

ctx := context.Background()
svc, err := services.NewQueueService(cfg)
if err != nil {
    return err
}
defer svc.Stop() // 若调用了 Start

job := jobs.NewExampleJob("hello")
if err := svc.Push(ctx, job); err != nil {
    return err
}
```

延迟：

```go
err := svc.Later(ctx, job, 5*time.Minute)
```

### 4. 启动消费者

**二选一**（不要重复消费同一队列）：

- **独立进程**：`go run ./cmd/worker`（或编译后的 `worker`），读取 `queue.queues` 与 Redis URL。
- **与应用同进程**：`queueService.Start()`（内部 `asynq.Server.Run`）；退出前 `queueService.Stop()`。

## 命令行工具

构建示例：`go build -o queue ./cmd/queue`、`go build -o worker ./cmd/worker`。

### `cmd/queue`

| 参数 | 说明 |
|------|------|
| `-config` | 配置文件路径，默认 `configs/config.yaml` |
| `-start` | 在本进程启动 Asynq Server，Ctrl+C 时优雅退出 |
| `-stop` | 调用 `QueueService.Stop()`；**每次运行均为新进程**，单独执行通常无正在运行的 Server，一般用于与 `-start` 同一次设计的扩展；独立 **`worker`** 请对进程发 **SIGINT/SIGTERM** |
| `-clear -queue=<name>` | 清空指定 Asynq 队列（Inspector 批量删除各状态任务） |
| `-list` | 列出配置中的队列名 |
| `-status` | 用 `Manager.Size` 查任务数；**`Active workers` 仅在本次进程调用过 `-start` 且未退出时为非零** |

```bash
go run ./cmd/queue -start
go run ./cmd/queue -clear -queue=default
go run ./cmd/queue -list
```

### `cmd/queue-status`

```bash
go run ./cmd/queue-status -all
go run ./cmd/queue-status -queue=default
```

输出为 Asynq **队列维度**的任务总数（含 pending / scheduled / active 等，与 `QueueInfo.Size` 一致）。

### `cmd/queue-test`

- 默认：Asynq 自动化用例（Push、Size、`Later`、`PushRaw`、`UniqueKey`、`Pop` 返回 `ErrPullNotSupported` 等）。
- `-seed`：向 `default` / `high` / `low` 写入示例任务，便于联调 `worker`。

### 监控（可选）

可使用 **[asynqmon](https://github.com/hibiken/asynqmon)** 连接同一 Redis，查看任务与队列状态。

## 生产环境

### systemd 示例（独立 worker）

```ini
[Unit]
Description=Go Admin Asynq Worker
After=network.target redis.service

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

### 实践建议

1. **连接**：默认只配 **`queue.connection.db`** 与顶层 `redis` 即可分库；必要时再用 `queue.connection.redis` 写完整 URL。
2. **并发**：通过各队列 `processes` 调节总 `Concurrency`。
3. **优雅退出**：依赖 Asynq `Shutdown`/`ShutdownTimeout`（来自 `queue.worker.timeout`）。
4. **可观测性**：队列长度用 `queue-status` 或 asynqmon；失败任务在 Asynq **归档** 中查看。

## 故障排除

| 现象 | 检查 |
|------|------|
| 任务不入队 | Redis 是否可达、`QueueRedisConnectionURL` 是否正确 |
| 任务不执行 | 是否启动 `worker` 或 `QueueService.Start()`；`job_type` 是否已 `RegisterAsynqHandlers` |
| 重复入队被拒 | 是否为 `ErrDuplicateJob`；Unique 窗口内 payload/类型是否相同 |
| `Handle` 未跑到 | 是否 import 了注册 `RegisterJobType` 的包；JSON 是否含正确 `job_type` |

## 相关文档

- [配置说明](../getting-started/configuration.md)
- [项目结构](../getting-started/structure.md)
- [部署指南](../deployment/README.md)
