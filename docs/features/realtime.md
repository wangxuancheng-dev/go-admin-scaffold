# Realtime channels (WebSocket / SSE)

实现库：WebSocket 使用 [`github.com/coder/websocket`](https://github.com/coder/websocket)（gorilla/websocket 已 archived，不再使用）。

## Auth

| Endpoint | Auth |
|----------|------|
| `POST /api/admin/v1/realtime/ticket` | `Authorization: Bearer <jwt>` → one-time ticket (Redis TTL, default 60s) |
| `GET /api/admin/v1/ws` | Bearer JWT, or `Sec-WebSocket-Protocol: access_token.<jwt>` / `ticket.<id>`, or `?ticket=<id>`; legacy `?token=<jwt>` only when `realtime.allow_query_token` (default **false** in production) |
| `POST /api/admin/v1/ws/join\|leave\|send` | `Authorization: Bearer <jwt>` |
| `GET /api/admin/v1/sse` | Same connect rules as WebSocket |
| `POST /api/admin/v1/sse/*` | `Authorization: Bearer <jwt>` |

服务端从不信任客户端自报的连接身份；`From` / 频道成员 ID 由 claims 或 ticket 解析出的用户决定。

WebSocket Origin 在 `Accept` 前按 `cors.allow_origins` 校验：生产禁止 `*`；无 Origin 的非浏览器客户端仍允许。协商到的子协议由 `coder/websocket` 的 `AcceptOptions.Subprotocols` 回显。

## Recommended connect flow

1. `POST /api/admin/v1/realtime/ticket` with Bearer access token  
2. Open WS/SSE with `?ticket=<id>` or subprotocol `ticket.<id>` (ticket is single-use and short-lived — safer than putting JWT in the URL)

## Connect examples

Bearer:

```http
GET /api/admin/v1/ws HTTP/1.1
Authorization: Bearer <access_token>
Connection: Upgrade
Upgrade: websocket
```

Ticket (preferred for browsers that cannot set Authorization on WS):

```http
POST /api/admin/v1/realtime/ticket
Authorization: Bearer <access_token>
```

```text
new WebSocket(url + "?ticket=" + ticket)
// or: new WebSocket(url, ["ticket." + ticket])
```

Legacy (dev only by default; disabled in production unless `realtime.allow_query_token: true`):

```text
ws://host/api/admin/v1/ws?token=<access_token>
```

## Join / Leave

```http
POST /api/admin/v1/ws/join?group_id=<id>
Authorization: Bearer <access_token>
```

## Implementation notes

| 组件 | 路径 |
|------|------|
| Accept / Origin | `internal/api/admin/handlers/ws_handler.go` |
| Hub / ReadPump / WritePump / Ping | `internal/core/ws/manager.go` |
| SSE | `internal/api/admin/handlers/sse_handler.go`（标准 HTTP，无第三方 WS 库） |
