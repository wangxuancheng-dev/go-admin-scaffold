# 项目结构说明

本文档说明 Go Admin Scaffold 的目录结构与扩展约定（与当前仓库对齐）。

## 目录结构

```
.
├── cmd/                    # 进程入口
│   ├── server/             # HTTP API（可选内嵌 cron）
│   ├── scheduler/          # 独立 cron 进程（生产推荐）
│   ├── worker/             # Asynq worker
│   ├── queue/              # 队列 CLI
│   ├── queue-status/       # 队列只读状态
│   ├── queue-test/         # 队列联调 / 用例
│   ├── tools/              # 运维工具（migrate / seed / make）
│   └── artisan/            # 交互式命令入口（make / migrate / seed / schedule:run）
├── configs/                # config.yaml / config.example.yaml
├── deploy/                 # Docker / supervisor 等
├── docs/                   # 文档（见 docs/README.md）
├── internal/
│   ├── api/                # admin / open HTTP
│   ├── bootstrap/          # SetupDB/Redis/Cache + NewContainer
│   ├── commands/           # artisan / tools 共用命令实现
│   ├── config/             # 配置加载与 Validate
│   ├── core/               # services / repositories / models / storage / metrics / ws / sse
│   ├── database/           # migrations / seeders
│   ├── routes/             # SetupRoutes(container)
│   └── schedule/           # cron kernel
├── locales/                # i18n YAML
├── pkg/                    # 可复用库（database, redis, response, queue, console, …）
├── scripts/
├── static/
├── storage/                # logs / uploads（运行时）
├── Dockerfile
├── go.mod
├── Makefile
└── README.md
```

> 模型在 `internal/core/models`；文件存储在 `internal/core/storage`；指标在 `internal/core/metrics`；依赖组装在 `internal/bootstrap/container.go`。

## CLI 分工

| 入口 | 用途 |
|------|------|
| `cmd/artisan` | 开发期统一命令：`make` / `migrate` / `seed` / `schedule:run` |
| `cmd/tools` | 运维脚本式调用（子命令在 `cmd/tools/commands`） |
| `cmd/scheduler` | 独立 cron；与 `scheduler.run_in_server=false` 配合 |
| `internal/commands` | 上述入口共享的命令实现 |

## 扩展开发

1. **组合根**：`SetupDatabase` / `SetupRedis` → `NewContainer(cfg, db, rdb)` → `routes.SetupRoutes`
2. **业务**：Repo 接口 + Service 构造注入；HTTP 为 `admin/v1` 结构体 Handler（依赖 `*ServiceAPI` 接口）
3. **中间件 / 可观测**：JWT/RBAC/OpLog 构造注入；`RateLimitRedis` fail-closed；Trace + `/metrics`
4. **调度**：开发可 `scheduler.run_in_server: true`；生产建议独立 `cmd/scheduler`
5. **存储**：仅 `internal/core/storage`

## 相关文档

- [文档索引](../README.md)
- [快速开始](quick-start.md)
- [配置说明](configuration.md)
- [架构与依赖注入](../advanced/architecture.md)
- [部署指南](../deployment/README.md)
