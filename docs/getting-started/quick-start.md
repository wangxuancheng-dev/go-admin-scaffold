# 快速开始

## 环境要求

- Go **1.23+**（与 `go.mod` 一致）
- MySQL 或 PostgreSQL（与 `configs/config.yaml` 中 `database` 段、`driver` 一致）
- Redis（缓存、队列、调度锁）

## 1. 获取代码与依赖

```bash
git clone <你的仓库地址> go-admin-scaffold
cd go-admin-scaffold
go mod download
```

## 2. 配置

```bash
cp configs/config.example.yaml configs/config.yaml
```

按环境修改 `configs/config.yaml`：**数据库连接、Redis、`jwt.secret`、`app.env` 等**。字段含义见 [配置说明](configuration.md)。

## 3. 数据库

创建库（名称与配置一致），例如：

```sql
CREATE DATABASE IF NOT EXISTS go_admin CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

迁移与填充：

```bash
go run ./cmd/tools/main.go migrate run
go run ./cmd/tools/main.go seed run
```

## 4. 启动

```bash
# HTTP API（端口见配置 app.port / server.address）
go run ./cmd/server/main.go
```

可选：队列消费（与内置 `QueueService` 二选一，避免重复消费同一队列）：

```bash
go run ./cmd/worker/main.go
```

## 5. 验证

- 健康检查：`GET /api/open/v1/public/health`  
  例：`curl http://localhost:8080/api/open/v1/public/health`
- 非 `production` 时可访问 Swagger：`/swagger/index.html`
- 队列自检：`go run ./cmd/queue-test`

默认账号以 seed 为准（常见为 `admin` / `admin123`，请以 `internal/database/seeders` 为准）。

## 常用命令（摘录）

| 目的 | 命令 |
|------|------|
| 迁移状态 | `go run ./cmd/tools/main.go migrate status` |
| 队列体量 | `go run ./cmd/queue-status -all` |
| 清空某队列 | `go run ./cmd/queue -clear -queue=default`（需配置） |
| Artisan | `go run ./cmd/artisan/main.go` |

更多子命令见 [命令行文档](../advanced/commands.md)。

## 下一步

- [目录结构](structure.md)  
- [配置说明](configuration.md)  
- [开发说明](../advanced/development.md)  
- [文档索引](../README.md)
