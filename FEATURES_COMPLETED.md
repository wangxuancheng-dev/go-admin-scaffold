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
- [x] WebSocket / SSE（Bearer / Sec-WebSocket-Protocol；query token 仅兼容；WS Origin 对齐 CORS）
- [x] Trace ID
- [x] `GET /metrics`（token 鉴权；status/method/latency）
- [x] `GET /api/open/v1/public/live` · `/ready`

### 基础设施
- [x] 组合根 DI：`bootstrap.NewContainer(cfg, db, rdb)`
- [x] 构造注入 Handler / 中间件（i18n 实例注入，无请求内 GetInstance）
- [x] Service → Repository 边界（User/Role/Menu/Todo 接口；Handler 依赖 *ServiceAPI）
- [x] Redis 限流 fail-closed（`RateLimitRedis`）
- [x] Asynq 队列 + worker / CLI
- [x] cron 调度 + Redis 锁（可 `scheduler.run_in_server=false` + `cmd/scheduler`）
- [x] 迁移 / seeder / artisan / tools
- [x] CLI：`database.WithContext` / `FromContext` + `Validate()`

### 测试与 CI / 文档
- [x] 核心包单测 + sqlite / miniredis 集成级测
- [x] GitHub Actions：vet、staticcheck、race、覆盖率门禁（infra≥65% 含 routes；services/repos≥20%）、Swagger drift、文档 sanity
- [x] 架构 / 测试 / Realtime / Tracing 文档

## 核心结构

```
cmd/           server, scheduler, worker, tools, artisan, queue*
internal/
  bootstrap/   Container + SetupDB/Redis/Cache
  api/admin/   v1 handlers, middleware, ws/sse
  api/open/    health (live/ready)
  core/        services, repositories, models, storage, metrics
  routes/      SetupRoutes(container)
pkg/           database, redis, response, queue, console, …
docs/          见 docs/README.md
```

## 默认账号（seed）

以 `internal/database/seeders` 为准；常见：`admin` / `admin123`。

---

**状态**: 可作二次开发底座  
**更新**: 2026-09
