# Realtime channels (WebSocket / SSE)

## Auth strategy

| Endpoint | Auth |
|----------|------|
| `GET /api/admin/v1/ws` | Query `token` (JWT). Identity is taken from JWT claims only. Optional `user_id` must match claim `username` or numeric `user_id` when present. |
| `POST /api/admin/v1/ws/join\|leave\|send` | `Authorization: Bearer <jwt>` (JWT middleware) |
| `GET /api/admin/v1/sse` | Same as WebSocket connect |
| `POST /api/admin/v1/sse/*` | `Authorization: Bearer <jwt>` |

Clients must not trust a client-supplied identity: the server always resolves the user via `AuthService.GetUserFromClaims`.

## Example connect

```text
ws://host/api/admin/v1/ws?token=<access_token>
# optional legacy: &user_id=<username_or_id>
```
