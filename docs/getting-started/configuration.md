# 配置说明

**以仓库内 `configs/config.example.yaml` 为权威模板**：复制为 `configs/config.yaml` 后按需修改。下列仅说明结构与代码中的用法。

## 配置文件

| 文件 | 作用 |
|------|------|
| `configs/config.yaml` | 本地/部署实际使用（勿提交密钥） |
| `configs/config.example.yaml` | 示例与字段说明 |

多环境可通过不同机器上的不同 `config.yaml`，或下方**环境变量**覆盖同一文件。

## 代码中加载

```go
cfg, err := config.LoadConfig()
if err != nil { /* ... */ }
if err := cfg.Validate(); err != nil { /* 生产环境强校验 */ }
```

显式文件（如队列 CLI）：`config.LoadConfigFromFile(path)`。

## 主要配置块（对应 YAML 顶层键）

| 块 | 说明 |
|----|------|
| `app` | 名称、环境 `env`（`production` 会收紧校验并关闭部分调试路由）、`port`、`debug` 等 |
| `server` | `address`（监听）、`mode`（Gin） |
| `database` | `driver`：`mysql` / `postgres`；连接字段与连接池 |
| `redis` | 主机、端口、密码、`db` |
| `jwt` | `secret`（生产至少 32 字符）、`expire_time`、`issuer` |
| `super_admin` | `user_ids`：超级管理员用户 ID 列表 |
| `log` | Zap + lumberjack：`level`、`filename`、`daily`、`timezone` 等 |
| `queue` | Asynq：`driver`、`queue`、`connection`、`worker`、`queues` — 详见 [队列文档](../features/queue.md) |
| `cache` | `driver`：`redis` / `file` 及 `options` |
| `storage` | `local` 或 `s3` |
| `cors` | 跨域；生产勿使用 `allow_origins: ["*"]` |
| `metrics` | `/metrics` 开关与 scrape token |
| `scheduler` | `run_in_server`：省略时非 production 默认 true、production 默认 false；可显式覆盖 |
| `realtime` | `allow_query_token`（生产默认 false）、`ticket_ttl_seconds`；配合 `POST /realtime/ticket` |
| `i18n` | 默认语言、加载路径、可用语言列表 |

## 环境变量覆盖

以下在 `internal/config` 中会从环境变量读取（未设置则用 YAML）：

`APP_NAME`、`APP_ENV`、`APP_MODE`、`APP_DEBUG`、`APP_URL`、`APP_PORT`、`APP_API_PREFIX`  
`JWT_SECRET`、`JWT_EXPIRE`、`JWT_ISSUER`  
`DB_DRIVER`、`DB_HOST`、`DB_PORT`、`DB_USERNAME`、`DB_PASSWORD`、`DB_DATABASE`、`DB_CHARSET`、`DB_SSLMODE`、`DB_TIMEZONE`、`DB_MAX_IDLE_CONNS`、`DB_MAX_OPEN_CONNS`、`DB_CONN_MAX_LIFETIME`  
`REDIS_HOST`、`REDIS_PORT`、`REDIS_PASSWORD`、`REDIS_DB`  
`CACHE_DRIVER`、`CACHE_PREFIX`  
`QUEUE_DRIVER`、`QUEUE_NAME` 等（队列其余见源码 `populateConfigFromViper`）  
`SERVER_ADDRESS`、`SERVER_MODE`  
`LOG_LEVEL`、`LOG_FILENAME`、`LOG_MAX_SIZE`、`LOG_MAX_BACKUPS`、`LOG_MAX_AGE`、`LOG_COMPRESS`、`LOG_DAILY`、`LOG_TIMEZONE`  
`I18N_DEFAULT_LOCALE`、`I18N_LOAD_PATH`  
`STORAGE_DRIVER`、`STORAGE_LOCAL_PATH`、`STORAGE_S3_*`

## 生产校验摘要

`config.Validate()` 在 `app.env=production` 时会检查：JWT 长度、数据库/Redis 非空、CORS 不使用 `*` 等。部署前务必调用。

## 相关文档

- [快速开始](quick-start.md)  
- [文档索引](../README.md)  
- [队列](../features/queue.md) · [缓存](../features/cache.md) · [部署](../deployment/README.md)
