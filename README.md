# Go Admin Scaffold

基于 **Gin、GORM、JWT、Redis、Asynq** 的后台 API 脚手架（用户/RBAC/日志/上传/队列/定时任务等）。

**完整文档目录（仅含仓库内真实存在的页面）见 [docs/README.md](docs/README.md)。**

## 功能速览

- 认证与 RBAC — [authentication](docs/features/authentication.md) · [rbac](docs/features/rbac.md)  
- 实时通道 — [realtime](docs/features/realtime.md)（WS/SSE JWT claims）  
- 队列（Asynq）— [queue](docs/features/queue.md)  
- 定时任务 — [scheduling](docs/features/scheduling.md)  
- 缓存 — [cache](docs/features/cache.md)  
- 架构 / DI — [architecture](docs/advanced/architecture.md) · 可观测 [tracing](docs/tracing.md)  
- API 约定 — [api/README.md](docs/api/README.md)  

## 快速开始（中文详版：[docs/getting-started/quick-start.md](docs/getting-started/quick-start.md)）

```bash
git clone <your-repo-url> && cd go-admin-scaffold
go mod download
cp configs/config.example.yaml configs/config.yaml
# 编辑 configs/config.yaml：database、redis、jwt.secret 等

go run ./cmd/tools/main.go migrate run
go run ./cmd/tools/main.go seed run
go run ./cmd/server/main.go
```

健康检查：`GET /api/open/v1/public/live` · `GET /api/open/v1/public/ready`  
队列自测：`go run ./cmd/queue-test`

## 部署与进阶

- [部署说明](docs/deployment/README.md)（二进制/Docker 等，与快速开始互补）  
- [配置与环境变量](docs/getting-started/configuration.md)  
- [测试与 CI](docs/advanced/testing.md) · [命令行](docs/advanced/commands.md)

## 许可

见仓库根目录 `LICENSE`。
