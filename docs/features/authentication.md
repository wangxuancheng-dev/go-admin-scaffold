# 用户认证指南

## 概述

本项目使用 JWT (JSON Web Token) 进行用户认证。

## 功能特性

- JWT 令牌认证
- 刷新令牌机制
- 多端登录控制
- 登录日志记录

## 配置说明

在 `configs/config.yaml` 中配置 JWT 相关参数：

```yaml
jwt:
  secret: your-secret-key
  expire: 24h  # Token 过期时间
  issuer: go-admin
```

## API 接口

### 登录接口

```http
POST /api/v1/auth/login
Content-Type: application/json

{
    "username": "admin",
    "password": "password"
}
```

### 刷新令牌

```http
POST /api/v1/auth/refresh
Authorization: Bearer <your-refresh-token>
```

## 使用示例

```go
// 在 API 处理器中获取当前用户
func (h *Handler) GetUserInfo(c *gin.Context) {
    user := auth.GetCurrentUser(c)
    // ... 处理逻辑
}
```

## 安全建议

1. 使用 HTTPS 传输
2. 定期轮换密钥
3. 设置合理的令牌过期时间
4. 实施登录失败次数限制（生产使用 Redis 限流：`RateLimitRedis`，多实例共享）

WebSocket / SSE 鉴权见 [realtime.md](realtime.md)。

## 常见问题

1. 令牌过期处理
2. 多设备登录控制
3. 密码重置流程

## 相关文档

- [API 文档](../api/README.md)
- [配置说明](../getting-started/configuration.md) 