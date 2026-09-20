# Tracing & Metrics

## Trace ID

全局中间件 `middleware.Trace()`（在 `cmd/server/setup` 最先挂载）为每个请求写入 `trace_id`。

- 响应 JSON 字段：`trace_id`（见 `pkg/response`）
- 可从 Header 传入上游 trace（实现见 `internal/api/admin/middleware/trace.go`）

## Metrics

进程级指标，无外部 Prometheus 客户端依赖：

```http
GET /metrics
Authorization: Bearer <metrics.token>   # or X-Metrics-Token
```

配置（`configs/config.example.yaml`）：

```yaml
metrics:
  enabled: true
  token: ""   # production 下若启用 metrics，token 必填
```

| 指标 | 类型 | 含义 |
|------|------|------|
| `go_admin_http_requests_total` | counter | 请求总数 |
| `go_admin_http_inflight` | gauge | 在途请求 |
| `go_admin_http_errors_total` | counter | HTTP status ≥ 500 |
| `go_admin_uptime_seconds` | gauge | 进程运行秒数 |
| `go_admin_http_requests_by_status_total` | counter | 按 status class（2xx/4xx/…） |
| `go_admin_http_requests_by_method_total` | counter | 按 HTTP method |
| `go_admin_http_request_duration_ms` | histogram | 请求耗时（毫秒） |

由 `internal/core/metrics` 中间件统计；`/metrics` 自身不计入。token 为空时（非生产）可不鉴权；生产必须配置 token。

## Health

| 路径 | 含义 |
|------|------|
| `/api/open/v1/public/live` | 进程存活 |
| `/api/open/v1/public/ready` | DB + Redis 就绪（失败 503） |

依赖通过 `open/v1.NewHealthHandler(db, rdb)` 注入。
