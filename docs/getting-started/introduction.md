# 项目介绍

**Go Admin Scaffold** 是基于 Go 的后台管理脚手架：REST API（Gin）、ORM（GORM）、JWT 认证、RBAC、操作日志、国际化、本地上传 / S3、Redis 缓存、**Asynq 队列**与 **cron 定时任务**（多实例可用 Redis 锁互斥）。

更细的模块说明见 [目录结构](structure.md) 与 [文档索引](../README.md)。

## 技术栈

| 类别 | 选型 |
|------|------|
| 语言 / 模块 | Go 1.23+，Go Modules |
| Web | Gin |
| 持久化 | GORM，MySQL / PostgreSQL（按配置） |
| 缓存 / 锁 / 队列后端 | Redis，Asynq |
| 认证 | JWT（`golang-jwt/jwt`） |
| 配置 | Viper（YAML） |
| 日志 | Zap |
| 定时 | `robfig/cron/v3` |
| API 文档 | Swag（开发环境 Swagger UI） |

## 常用入口

1. [快速开始](quick-start.md) — 跑通服务与数据库  
2. [配置说明](configuration.md) — 改 `configs/config.yaml`  
3. [队列说明](../features/queue.md) — `worker` 与 `queue-test`  
4. [定时任务](../features/scheduling.md) — `internal/schedule` 与 `Unique` 任务  

## 仓库与许可

- 源码与 Issue：以你托管平台为准（如 GitHub 上的本项目仓库）  
- 许可证：见仓库根目录 `LICENSE`（常见为 MIT）
