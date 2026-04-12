# 配置说明

本文档详细说明了 Go Admin Scaffold 的配置选项和使用方法。

## 配置文件

项目使用 YAML 格式的配置文件，主要配置文件位于 `configs/` 目录：

- `config.yaml` - 主配置文件
- `config.example.yaml` - 配置示例文件
- `config.dev.yaml` - 开发环境配置
- `config.prod.yaml` - 生产环境配置

## 配置结构

### 1. 应用配置 (app)

```yaml
app:
  name: "Go Admin"           # 应用名称
  env: "development"         # 运行环境：development, production, testing
  debug: true               # 调试模式
  port: 8080                # 服务端口
  jwt_secret: "your-secret" # JWT密钥
  jwt_expire: 24h           # JWT过期时间
  timezone: "Asia/Shanghai" # 时区设置
  locale: "zh_CN"           # 默认语言
```

### 2. 数据库配置 (mysql)

```yaml
mysql:
  host: "localhost"         # 数据库主机
  port: 3306               # 数据库端口
  username: "root"         # 数据库用户名
  password: "password"     # 数据库密码
  database: "go_admin"     # 数据库名称
  charset: "utf8mb4"       # 字符集
  max_idle_conns: 10       # 最大空闲连接数
  max_open_conns: 100      # 最大打开连接数
  conn_max_lifetime: 1h    # 连接最大生命周期
```

### 3. Redis配置 (redis)

```yaml
redis:
  host: "localhost"         # Redis主机
  port: 6379               # Redis端口
  password: ""             # Redis密码
  db: 0                    # 数据库索引
  pool_size: 10            # 连接池大小
  min_idle_conns: 5        # 最小空闲连接数
  dial_timeout: 5s         # 连接超时时间
  read_timeout: 3s         # 读取超时时间
  write_timeout: 3s        # 写入超时时间
```

### 4. 日志配置 (logger)

```yaml
logger:
  level: "debug"           # 日志级别：debug, info, warn, error
  filename: "logs/app.log" # 日志文件路径
  max_size: 100           # 单个日志文件最大大小(MB)
  max_age: 30             # 日志文件保留天数
  max_backups: 10         # 最大保留文件数
  compress: true          # 是否压缩
  json_format: false      # 是否使用JSON格式
```

### 5. 队列配置 (queue)

后端为 **[Asynq](https://github.com/hibiken/asynq)**（Redis）。`driver` 填 `redis` 或 `asynq` 均可。

```yaml
queue:
  driver: "redis"
  queue: "default"
  unique_ttl: 0                 # 秒；Unique 窗口，0 用驱动默认
  connection:
    # 不写 redis 时：与顶层 redis 共用 host/port/password，仅用 db 指定逻辑库（可与默认 redis.db 不同）
    db: 1
    # 需要完整 URL 时再写：redis: "redis://:pass@host:6379/2"
  worker:
    timeout: 60                 # Asynq Server ShutdownTimeout（秒）
  queues:
    default:
      priority: 3
      processes: 1
    high:
      priority: 6
      processes: 2
    low:
      priority: 1
      processes: 1
```

详见 [队列功能说明](../features/queue.md)。

### 6. 存储配置 (storage)

```yaml
storage:
  default: "local"         # 默认存储驱动：local, s3
  disks:
    local:
      driver: "local"      # 本地存储驱动
      root: "storage/app"  # 存储根目录
      url: "http://localhost:8080/storage" # 访问URL
    s3:
      driver: "s3"         # S3存储驱动
      key: ""              # AWS Access Key
      secret: ""           # AWS Secret Key
      region: "us-east-1"  # AWS Region
      bucket: ""           # S3 Bucket
      url: ""              # 自定义域名
```

### 7. 缓存配置 (cache)

```yaml
cache:
  default: "redis"         # 默认缓存驱动：redis, memory
  stores:
    redis:
      driver: "redis"      # Redis缓存驱动
      prefix: "cache:"     # 缓存键前缀
    memory:
      driver: "memory"     # 内存缓存驱动
      prefix: "cache:"     # 缓存键前缀
```

## 环境变量

项目支持通过环境变量覆盖配置文件中的设置：

```bash
# 应用配置
APP_NAME=GoAdmin
APP_ENV=production
APP_DEBUG=false
APP_PORT=8080
APP_JWT_SECRET=your-secret
APP_TIMEZONE=Asia/Shanghai

# 数据库配置
DB_HOST=localhost
DB_PORT=3306
DB_USERNAME=root
DB_PASSWORD=password
DB_DATABASE=go_admin

# Redis配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# 其他配置
QUEUE_CONNECTION=redis
STORAGE_DISK=s3
CACHE_DRIVER=redis
```

## 配置使用

### 1. 获取配置

```go
import "github.com/1768177868/go-admin-scaffold/pkg/config"

// 获取配置
port := config.Get("app.port")
dbHost := config.Get("mysql.host")

// 获取带默认值的配置
timeout := config.Get("redis.timeout", 60) // 默认60秒

// 获取环境变量
env := config.Get("app.env", "development")
```

### 2. 动态配置

```go
// 设置配置
config.Set("app.name", "New Name")

// 检查配置是否存在
if config.Has("app.debug") {
    // ...
}
```

## 最佳实践

### 1. 环境配置

- 开发环境：使用 `config.dev.yaml`
- 测试环境：使用 `config.test.yaml`
- 生产环境：使用 `config.prod.yaml`
- 敏感信息：使用环境变量

### 2. 安全建议

- 生产环境禁用调试模式
- 使用强密码和密钥
- 定期轮换密钥
- 限制数据库访问
- 使用环境变量存储敏感信息

### 3. 性能优化

- 合理设置连接池大小
- 配置适当的超时时间
- 使用缓存减少数据库访问
- 根据需求调整队列配置

### 4. 维护建议

- 保持配置文件版本控制
- 记录配置变更历史
- 定期检查配置有效性
- 备份重要配置

## 常见问题

### 1. 配置不生效

检查：
- 配置文件路径是否正确
- 环境变量是否正确设置
- 配置项名称是否正确
- 配置格式是否正确

### 2. 环境变量问题

检查：
- 环境变量名称是否正确
- 环境变量是否已加载
- 环境变量值是否正确
- 是否有权限访问

### 3. 配置冲突

解决：
- 检查配置文件优先级
- 确认环境变量覆盖
- 验证配置加载顺序
- 检查配置合并逻辑

## 相关文档

- [快速开始指南](quick-start.md)
- [项目结构说明](structure.md)
- [开发环境配置](../advanced/development.md)
- [部署指南](../deployment/README.md) 