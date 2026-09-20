# Realtime channels (WebSocket / SSE)

## Auth

| Endpoint | Auth |
|----------|------|
| `GET /api/admin/v1/ws?token=<jwt>` | Identity from JWT claims only |
| `POST /api/admin/v1/ws/join\|leave\|send` | `Authorization: Bearer <jwt>`；join/leave 的身份取自 JWT，仅需 `group_id` |
| `GET /api/admin/v1/sse?token=<jwt>` | 同 WebSocket |
| `POST /api/admin/v1/sse/*` | `Authorization: Bearer <jwt>` |

服务端从不信任客户端自报的连接身份；`From` / 频道成员 ID 由 claims 解析出的用户决定。

## Connect

```text
ws://host/api/admin/v1/ws?token=<access_token>
```

```text
GET /api/admin/v1/sse?token=<access_token>
```

## Join / Leave

```http
POST /api/admin/v1/ws/join?group_id=<id>
Authorization: Bearer <access_token>
```
