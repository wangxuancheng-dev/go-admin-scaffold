# 架构与依赖注入

## 组合根

启动顺序（`cmd/server/setup`）：

1. `config.LoadConfig` + `Validate`
2. logger / `SetupDatabase` → `*gorm.DB` / `SetupRedis` → `*redis.Client` / `SetupCache(cfg, rdb)` / `i18n.Init`
3. `bootstrap.NewContainer(cfg, db, rdb)` — **只组装一次**
4. `routes.SetupRoutes(engine, container)` — Handler / 中间件构造注入（含 `middleware.I18n(i18n.Instance())`）

```text
SetupDB/Redis → NewContainer → AdminAPI / JWT / RBAC / OpLog / Upload / Health
```

HTTP 路径一律构造注入，不按请求 `New` Service。Handler 依赖 `UserServiceAPI` / `RoleServiceAPI` / `MenuServiceAPI` 等接口。

## 分层

| 层 | 路径 |
|----|------|
| HTTP | `internal/api/admin/v1`（结构体 Handler） |
| Realtime | `internal/api/admin/handlers`（WS / SSE） |
| 中间件 | `internal/api/admin/middleware` |
| 服务 | `internal/core/services` |
| 仓储 | `internal/core/repositories`（含 RBAC） |
| 模型 | `internal/core/models` |
| 存储 | `internal/core/storage` |

扩展业务：在 Container 接线 → 新增 Handler 方法 → `router.go` 挂路由。

## 调度

- 非 production：`cmd/server` 默认内嵌 cron（`ApplyEnvDefaults`）
- production：默认 `scheduler.run_in_server=false`，另起 `cmd/scheduler`
- Realtime：production 默认禁 `?token=` JWT；用 Bearer / 子协议 / `POST /realtime/ticket`

## CLI

```go
db, _ := bootstrap.SetupDatabase(cfg)
ctx := database.WithContext(context.Background(), db)
manager.RunFromArgsWithContext(ctx)
// 命令内：database.FromContext(ctx)
```

`cmd/artisan` 与 `cmd/tools` 共用 `internal/commands`；开发用 artisan，运维脚本可用 tools。

## 可观测

- Trace：`middleware.Trace()` → 响应 `trace_id`
- Metrics：`GET /metrics`（Prometheus 文本，进程计数）
- Health：`/api/open/v1/public/live` · `/ready`

## Module

```text
module go-admin-scaffold
```

引用示例：`import "go-admin-scaffold/internal/core/services"`。
