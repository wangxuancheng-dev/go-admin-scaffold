# 快速开始指南

本指南将帮助您快速搭建和运行 Go Admin Scaffold 项目。

## 环境要求

- Go 1.18 或更高版本
- MySQL 5.7 或更高版本
- Redis 6.0 或更高版本
- Git

## 安装步骤

### 1. 获取代码

```bash
# 克隆项目
git clone https://github.com/1768177868/go-admin-scaffold.git
cd go-admin-scaffold

# 安装依赖
go mod download
```

### 2. 配置环境

1. 复制配置文件：
```bash
cp configs/config.example.yaml configs/config.yaml
```

2. 修改配置文件 `configs/config.yaml`：
```yaml
# 数据库配置
mysql:
  host: "localhost"
  port: 3306
  username: "root"
  password: "your_password"
  database: "go_admin"

# Redis 配置
redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0

# 应用配置
app:
  name: "Go Admin"
  env: "development"
  debug: true
  port: 8080
  jwt_secret: "your-secret-key"
```

### 3. 初始化数据库

1. 创建数据库：
```sql
CREATE DATABASE IF NOT EXISTS go_admin CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

2. 运行数据库迁移：
```bash
# 运行所有迁移
go run cmd/tools/main.go migrate run

# 运行数据填充
go run cmd/tools/main.go seed run
```

### 4. 启动服务

1. 启动主服务：
```bash
# 开发模式
go run cmd/server/main.go

# 或使用编译后的二进制文件
./server.exe
```

2. 启动队列服务（可选）：
```bash
# 开发模式
go run cmd/worker/main.go

# 或使用编译后的二进制文件
./worker.exe
```

## 验证安装

### 1. 访问服务

打开浏览器访问：`http://localhost:8080`

### 2. 默认账户

系统初始化后会创建以下默认账户：

**管理员账户：**
- 用户名：`admin`
- 密码：`admin123`
- 角色：管理员（所有权限）

**测试账户：**
- 用户名：`manager` / `user`
- 密码：`admin123`
- 角色：经理 / 普通用户（有限权限）

### 3. 检查服务状态

```bash
# 检查主服务状态
curl http://localhost:8080/api/v1/health

# 检查队列状态（如果启用了队列服务）
./queue-status.exe -all
```

## 开发模式

### 1. 热重载（开发环境）

使用 `air` 工具实现热重载：

```bash
# 安装 air
go install github.com/cosmtrek/air@latest

# 启动带热重载的服务
air
```

### 2. 调试模式

```bash
# 使用调试模式启动服务
go run cmd/server/main.go -debug

# 或使用编译后的二进制文件
./server.exe -debug
```

### 3. 查看日志

```bash
# 查看应用日志
tail -f logs/app.log

# 查看队列日志（如果启用了队列服务）
tail -f logs/queue.log
```

## 常用命令

### 数据库管理

```bash
# 查看迁移状态
go run cmd/tools/main.go migrate status

# 回滚最后一次迁移
go run cmd/tools/main.go migrate rollback

# 重置所有迁移
go run cmd/tools/main.go migrate reset

# 刷新迁移（重置并重新运行）
go run cmd/tools/main.go migrate refresh
```

### 队列管理

```bash
# 启动 Asynq 消费端（与 go run ./cmd/worker 二选一，勿重复消费）
go run ./cmd/queue -start

# 或独立 worker
go run ./cmd/worker

# 查看队列任务规模（Asynq 各状态合计）
go run ./cmd/queue-status -all

# 清空指定队列
go run ./cmd/queue -clear -queue=default
```

### 开发工具

```bash
# 生成新的控制器
go run cmd/tools/main.go make controller user

# 生成新的模型
go run cmd/tools/main.go make model user

# 生成新的迁移
go run cmd/tools/main.go make migration create_users_table

# 生成新的数据填充
go run cmd/tools/main.go make seeder user
```

## 下一步

- 查看 [项目结构说明](structure.md) 了解项目组织
- 阅读 [配置指南](configuration.md) 了解详细配置
- 参考 [开发指南](../advanced/development-guide.md) 开始开发
- 查看 [API 文档](../api/README.md) 了解接口使用

## 常见问题

### 1. 数据库连接失败

检查：
- 数据库服务是否运行
- 数据库配置是否正确
- 数据库用户权限

### 2. Redis 连接失败

检查：
- Redis 服务是否运行
- Redis 配置是否正确
- Redis 连接是否可用

### 3. 服务启动失败

检查：
- 端口是否被占用
- 配置文件是否正确
- 日志文件中的错误信息

### 4. 队列服务问题

检查：
- 队列服务是否启动
- 队列配置是否正确
- 队列日志中的错误信息

## 获取帮助

- 查看 [常见问题](../faq/README.md)
- 提交 [Issue](https://github.com/1768177868/go-admin-scaffold/issues)
- 加入技术交流群
- 发送邮件至：your-email@example.com 