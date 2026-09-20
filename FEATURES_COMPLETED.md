# Go Admin Scaffold — 功能完成情况

后端 API 脚手架（Gin + GORM + JWT + Redis + Asynq）。前端不在本仓库。

文档索引：[docs/README.md](docs/README.md)

## 已完成

### 认证与授权
- [x] JWT 登录 / 登出 / Refresh
- [x] 验证码
- [x] RBAC（角色 / 菜单权限码）+ Redis 权限缓存
- [x] 超级管理员配置（`super_admin.user_ids`）

### 业务 API
- [x] 用户 / 角色 / 菜单 CRUD
- [x] 登录日志 / 操作日志
- [x] 个人资料
- [x] 文件上传（Local / S3；UUID 文件名；JWT 保护本地文件）
- [x] Todo 示例 CRUD
- [x] i18n locales / translations

### 实时与可观测
- [x] WebSocket（`coder/websocket`）/ SSE（结构化日志；Bearer / 子协议 / ticket；生产默认禁 `?token=`）
- [x] `POST /realtime/ticket` 短时一次性连接票
- [x] Trace ID
- [x] `GET /metrics`（token 鉴权；status/method/latency）
- [x] `GET /api/open/v1/public/live` · `/ready`

### 基础设施
- [x] 组合根 DI：`NewContainerWithCache` + Cache / RealtimeTicket 注入
- [x] 构造注入 Handler / 中间件（i18n 实例注入）
- [x] Service → Repository 边界（User/Role/Menu/Todo *ServiceAPI）
- [x] Redis 限流 fail-closed（`RateLimitRedis`）
- [x] Asynq 队列（`internal/core/queuesvc`）+ worker / CLI
- [x] cron：生产默认不内嵌；`cmd/scheduler` 独立进程
- [x] 迁移 / seeder / artisan / tools
- [x] CLI：`database.WithContext` / `FromContext` + `Validate()` + `ApplyEnvDefaults`

### 测试与 CI / 文档
- [x] 核心包单测 + sqlite / miniredis 集成级测
- [x] GitHub Actions：vet、staticcheck、race、infra≥63%、services/repos≥70%、Swagger drift、文档 sanity
- [x] 架构 / 测试 / Realtime / Tracing 文档

## 核心结构

```
cmd/           server, scheduler, worker, tools, artisan, queue*
internal/
  bootstrap/   Container + SetupDB/Redis/Cache
  api/admin/   v1 handlers, middleware, ws/sse
  api/open/    health (live/ready)
  core/        services, repositories, queuesvc, models, storage, metrics
  routes/      SetupRoutes(container)
pkg/           database, redis, response, queue, console, …
docs/          见 docs/README.md
```

## 默认账号（seed）

以 `internal/database/seeders` 为准；常见：`admin` / `admin123`。

---

**状态**: 可作二次开发底座（生产路径已硬化）  
**更新**: 2026-09
