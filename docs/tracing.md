# Tracing & Metrics

## Trace ID

全局中间件 `middleware.Trace()`（在 `cmd/server/setup` 最先挂载）为每个请求写入 `trace_id`。

- 响应 JSON 字段：`trace_id`（见 `pkg/response`）
- 可从 Header 传入上游 trace（实现见 `internal/api/admin/middleware/trace.go`）

## Metrics

进程级指标，无额外依赖：

```http
GET /metrics
```

| 指标 | 类型 | 含义 |
|------|------|------|
| `go_admin_http_requests_total` | counter | 请求总数 |
| `go_admin_http_inflight` | gauge | 在途请求 |
| `go_admin_http_errors_total` | counter | HTTP status ≥ 500 |
| `go_admin_uptime_seconds` | gauge | 进程运行秒数 |

由 `internal/core/metrics` 中间件统计；`/metrics` 自身不计入。

生产可对接 Prometheus scrape；需要直方图/多维标签时可再换正式 `client_golang`。

## Health

| 路径 | 含义 |
|------|------|
| `/api/open/v1/public/live` | 进程存活 |
| `/api/open/v1/public/ready` | DB + Redis 就绪（失败 503） |

依赖通过 `open/v1.NewHealthHandler(db, rdb)` 注入。
