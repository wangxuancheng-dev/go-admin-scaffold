# 文档索引

面向本仓库 **实际存在的文档**，按主题归类；详细说明点进对应链接即可。

## 入门

| 文档 | 说明 |
|------|------|
| [项目介绍](getting-started/introduction.md) | 定位与技术栈 |
| [快速开始](getting-started/quick-start.md) | 克隆、配置、迁移、启动、常用命令 |
| [目录结构](getting-started/structure.md) | `cmd/`、`internal/`、`pkg/` 等 |
| [配置说明](getting-started/configuration.md) | `config.yaml` 字段说明 |

## 功能

| 文档 | 说明 |
|------|------|
| [认证](features/authentication.md) | JWT、登录等 |
| [RBAC](features/rbac.md) | 角色与权限 |
| [Realtime](features/realtime.md) | WebSocket（coder/websocket）/ SSE 鉴权约定 |
| [队列（Asynq）](features/queue.md) | 入队、`worker`、CLI、`queue-test` |
| [定时任务](features/scheduling.md) | cron、多实例与 Redis 锁 |
| [缓存](features/cache.md) | Redis / 文件缓存 |

## 开发与运维

| 文档 | 说明 |
|------|------|
| [数据库迁移](database/migrations.md) | 迁移与约定 |
| [API 说明](api/README.md) | 路径前缀、响应格式；Swagger 在非 `production` 下 `/swagger/*` |
| [部署](deployment/README.md) | 二进制 / Docker 等思路 |
| [命令行（artisan）](advanced/commands.md) | 自定义命令、注册方式 |
| [开发说明](advanced/development.md) | 本地开发习惯 |
| [架构与 DI](advanced/architecture.md) | 组合根、分层、CLI、可观测 |
| [测试](advanced/testing.md) | 测试约定与 CI |
| [日志（Zap）](logger.md) | 日志配置 |
| [Tracing / Metrics](tracing.md) | 请求追踪与 `/metrics` |

## 示例

- [Todo CRUD](examples/todo-crud.md)
- 队列自测：`go run ./cmd/queue-test`（`-seed` 往多队列写示例任务）

## 维护

文档与代码不一致时，以仓库源码为准；欢迎直接改对应 `.md` 并提 PR。
