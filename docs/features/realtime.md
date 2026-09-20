# Realtime channels (WebSocket / SSE)

## Auth

| Endpoint | Auth |
|----------|------|
| `GET /api/admin/v1/ws` | Prefer `Authorization: Bearer <jwt>`，或 `Sec-WebSocket-Protocol: access_token.<jwt>`；兼容 `?token=<jwt>`（易进 access log，不推荐） |
| `POST /api/admin/v1/ws/join\|leave\|send` | `Authorization: Bearer <jwt>`；join/leave 身份取自 JWT，仅需 `group_id` |
| `GET /api/admin/v1/sse` | Prefer `Authorization: Bearer <jwt>`；兼容 `?token=<jwt>` |
| `POST /api/admin/v1/sse/*` | `Authorization: Bearer <jwt>` |

服务端从不信任客户端自报的连接身份；`From` / 频道成员 ID 由 claims 解析出的用户决定。

WebSocket `CheckOrigin` 使用 `cors.allow_origins`：生产禁止 `*`，浏览器必须匹配白名单；无 Origin 的非浏览器客户端仍允许。若协商了 `Sec-WebSocket-Protocol`，升级响应会回显所选子协议。

## Connect

推荐（Bearer）：

```http
GET /api/admin/v1/ws HTTP/1.1
Authorization: Bearer <access_token>
Connection: Upgrade
Upgrade: websocket
```

或子协议（浏览器 `WebSocket` 构造函数第二参数）：

```text
new WebSocket(url, ["access_token." + accessToken])
```

遗留（不推荐）：

```text
ws://host/api/admin/v1/ws?token=<access_token>
```

SSE：

```http
GET /api/admin/v1/sse
Authorization: Bearer <access_token>
```

## Join / Leave

```http
POST /api/admin/v1/ws/join?group_id=<id>
Authorization: Bearer <access_token>
```
