# 用户认证指南

## 概述

本项目使用 JWT（JSON Web Token）进行用户认证。

## 功能特性

- JWT 登录 / 登出 / Refresh
- 验证码
- 登录日志
- Redis 限流（登录、验证码、Refresh）

## 配置说明

在 `configs/config.yaml` 中配置（字段以 `configs/config.example.yaml` 为准）：

```yaml
jwt:
  secret: "change-me-use-at-least-32-chars-in-production"
  expire_time: 86400  # seconds
  issuer: go-admin
```

生产环境（`app.env=production`）要求 `jwt.secret` 至少 32 字符。

## API 接口

前缀为 **`/api/admin/v1`**。

### 登录

```http
POST /api/admin/v1/auth/login
Content-Type: application/json

{
    "username": "admin",
    "password": "admin123",
    "captcha_id": "...",
    "captcha_code": "..."
}
```

### 刷新令牌

```http
POST /api/admin/v1/auth/refresh
Authorization: Bearer <access-token>
```

### 登出

```http
POST /api/admin/v1/auth/logout
Authorization: Bearer <access-token>
```

### 验证码

```http
GET /api/admin/v1/auth/captcha
```

## 在 Handler 中取当前用户

中间件 `JWT` 将用户写入 Gin Context（键名 `user`）。可用：

```go
userVal, ok := c.Get("user")
if !ok {
    // unauthorized
}
user := userVal.(*models.User)
```

## 安全建议

1. 使用 HTTPS
2. 定期轮换密钥
3. 设置合理的令牌过期时间
4. 生产使用 Redis 限流（`RateLimitRedis`，多实例共享）
5. WebSocket / SSE 通过 query `token` 传 JWT；浏览器 Origin 受 `cors.allow_origins` 约束

WebSocket / SSE 鉴权见 [realtime.md](realtime.md)。

## 相关文档

- [API 文档](../api/README.md)
- [配置说明](../getting-started/configuration.md)
- [RBAC](rbac.md)
