# 项目结构说明

本文档详细说明了 Go Admin Scaffold 项目的目录结构和各个模块的作用。

## 目录结构

```
.
├── cmd/                    # 应用程序入口
│   ├── server/            # 主服务入口
│   ├── worker/            # 队列工作进程
│   ├── queue/             # 队列管理工具
│   ├── queue-status/      # 队列状态工具
│   ├── queue-test/        # 队列测试（默认跑用例；-seed 写入示例任务）
│   ├── migrate/           # 数据库迁移工具
│   ├── tools/             # 通用工具
│   └── artisan/           # 命令行工具
├── configs/               # 配置文件
│   ├── config.yaml        # 主配置文件
│   └── config.example.yaml # 配置示例
├── deploy/                # 部署相关文件
│   ├── docker/           # Docker 配置
│   └── scripts/          # 部署脚本
├── docs/                  # 项目文档
│   ├── api/              # API 文档
│   ├── features/         # 功能文档
│   └── getting-started/  # 入门文档
├── internal/              # 内部代码
│   ├── api/              # API 实现
│   ├── bootstrap/        # 应用启动
│   ├── commands/         # 命令实现
│   ├── config/           # 配置管理
│   ├── core/             # 核心功能
│   ├── database/         # 数据库相关
│   ├── models/           # 数据模型
│   ├── routes/           # 路由定义
│   └── schedule/         # 任务调度
├── locales/              # 国际化文件
├── pkg/                  # 公共库
│   ├── cache/            # 缓存工具
│   ├── console/          # 命令行工具
│   ├── database/         # 数据库工具
│   ├── i18n/             # 国际化工具
│   ├── locker/           # 分布式锁
│   ├── logger/           # 日志工具
│   ├── queue/            # 队列工具
│   ├── response/         # 响应工具
│   ├── utils/            # 通用工具
│   └── validate/         # 校验工具
├── scripts/              # 脚本文件
├── static/               # 静态文件
├── storage/              # 存储目录
│   ├── app/             # 应用存储
│   ├── logs/            # 日志文件
│   └── uploads/         # 上传文件
├── .gitignore           # Git 忽略文件
├── Dockerfile           # Docker 构建文件
├── go.mod              # Go 模块文件
├── go.sum              # Go 依赖版本锁定
├── Makefile            # 构建脚本
└── README.md           # 项目说明
```

> 说明：文件存储实现位于 `internal/core/storage`（Local/S3），不再使用 `pkg/storage`。数据模型在 `internal/core/models`。

## 核心目录说明

### 1. cmd/ - 应用程序入口

#### server/
- `main.go`: 主服务入口文件
- 负责启动 HTTP 服务器
- 加载配置和初始化服务

#### worker/
- `main.go`: 独立 **Asynq Server**（读取 `queue.queues` 与 Redis URL，与业务中 `QueueService.Start()` 二选一部署）

#### queue/
- `main.go`: 队列 CLI（`-start` 启动 Asynq Server、`-clear`、`-list`、`-status` 等）

#### queue-status/
- 只读查看各 Asynq 队列任务总数（`QueueInfo.Size`）

#### queue-test/
- 默认：Asynq 队列自动化用例（Push、Size、`Later`、`PushRaw`、`UniqueKey`、`Pop` 不支持等）
- `go run ./cmd/queue-test -seed`：向 default/high/low 写入示例任务，便于联调 `worker`

#### migrate/
- 数据库迁移工具
- 管理数据库版本
- 执行数据库迁移

#### tools/
- 通用工具集
- 开发辅助工具
- 系统维护工具

#### artisan/
- 命令行工具框架
- 自定义命令支持
- 代码生成工具

### 2. internal/ - 内部代码

#### api/
- API 控制器
- 请求处理
- 响应封装

#### bootstrap/
- 应用启动配置
- 服务初始化
- 依赖注入

#### commands/
- 命令实现
- 命令行工具
- 任务处理

#### config/
- 配置管理
- 环境变量
- 配置加载

#### core/
- 核心业务逻辑
- 服务实现
- 功能模块

#### database/
- 数据库操作
- 迁移管理
- 数据填充

#### models/
- 数据模型
- 业务实体
- 数据验证

#### routes/
- 路由定义
- 中间件配置
- API 分组

#### schedule/
- 定时任务
- 任务调度
- 计划任务

### 3. pkg/ - 公共库

#### cache/
- Redis 缓存
- 内存缓存
- 缓存接口

#### console/
- 命令行工具
- 命令定义
- 参数解析

#### database/
- 数据库连接
- 查询构建
- 事务管理

#### i18n/
- 国际化支持
- 多语言管理
- 翻译工具

#### locker/
- 分布式锁
- 并发控制
- 锁管理

#### logger/
- 日志配置
- 日志记录
- 日志轮转

#### queue/
- 队列接口与 `Manager`（Asynq 客户端 / Inspector）

#### response/
- 响应格式
- 状态码
- 错误响应

#### storage/
- 文件存储
- 存储接口
- 文件管理

#### utils/
- 字符串工具
- 时间工具
- 加密工具
- 其他工具

### 4. storage/ - 存储目录

#### app/
- 应用数据
- 缓存文件
- 临时文件

#### logs/
- 应用日志
- 错误日志
- 访问日志

#### uploads/
- 用户上传
- 图片文件
- 文档文件

## 扩展开发（路径以本仓库为准）

1. **组合根**：`module go-admin-scaffold`；`SetupDatabase/SetupRedis` 返回句柄 → `NewContainer(cfg, db, rdb)`；Cache 注入 Redis client。
2. **业务**：Repo（含 RBAC）+ Service 构造注入；HTTP 为 `admin/v1` 结构体 Handler。
3. **中间件 / 可观测**：JWT/RBAC/OpLog 构造注入；`RateLimitRedis` fail-closed；`GET /metrics` + Trace。
4. **CLI**：`database.WithContext` / `FromContext`；tools/artisan/scheduler 显式传 DB。
5. **存储**：仅 `internal/core/storage`。




## 相关文档

- [文档索引](../README.md)
- [快速开始](quick-start.md)
- [配置说明](configuration.md)
- [部署指南](../deployment/README.md) 