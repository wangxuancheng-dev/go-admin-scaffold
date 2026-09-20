# 测试指南

与当前仓库实现对齐的测试约定（`module go-admin-scaffold`）。

## 怎么跑

```bash
# 全量
go test ./...

# 核心包覆盖率（与 CI 门禁包一致）
go test -coverprofile=coverage.out -covermode=atomic \
  ./internal/bootstrap \
  ./internal/core/metrics \
  ./internal/core/middleware \
  ./internal/core/storage \
  ./internal/api/open/v1 \
  ./internal/api/admin/handlers \
  ./pkg/utils \
  ./pkg/response \
  ./pkg/ginext

go tool cover -func=coverage.out
```

CI（`.github/workflows/ci.yml`）还会：

1. `go mod verify` + `go vet ./...`
2. 全量 `go test ./...`
3. 上表核心包覆盖率门槛（≥ 55%）
4. 文档完整性检查
5. Swagger 产物漂移检查（`swag init` 后 `git diff --exit-code`）

## 策略

| 层级 | 做法 | 示例 |
|------|------|------|
| 单元 | mock / stub 接口 | `UserServiceAPI` mock、`RBACPermissionStore` 内存实现 |
| HTTP | `httptest` + Gin TestMode | `admin/v1` Handler、JWT/RBAC、WS/SSE |
| 集成级 | 纯 Go sqlite（glebarez）/ miniredis | `RBACRepository`、`RateLimitRedis`、Role CRUD |
| 组合根 | 空 `*gorm.DB` 仅验证接线 | `bootstrap.NewContainer` |

测试里直接 `NewXxxHandler(deps)` 或 `middleware.JWT(authSvc)`；Realtime join/leave 需在 context 里 `Set("user", *models.User)`。

## Handler 单测模板

```go
package v1_test

import (
	adminv1 "go-admin-scaffold/internal/api/admin/v1"

	"github.com/gin-gonic/gin"
)

// Mock 实现 services.UserServiceAPI，再：
// h := adminv1.NewUserHandler(mockSvc, nil)
// r.GET("/users", h.ListUsers)
```

参考：

- `internal/api/admin/v1/user_test.go`
- `internal/api/admin/middleware/jwt_rbac_test.go`
- `internal/api/admin/handlers/handlers_test.go`

## 仓储集成级

```go
db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
// AutoMigrate 后 repositories.NewRBACRepository(db)
```

## 限流 / Redis

```go
mr, _ := miniredis.Run()
rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
// coremiddleware.RateLimitRedis(rdb, time.Minute, 2)
```

## 约定

1. 表驱动优先；断言用 `testify`。
2. 业务错误码看 JSON `code`（HTTP 多为 200）。
3. 构造 `*config.Config`，不读真实 `configs/config.yaml`。
4. `pkg/redis.Setup` / `pkg/database.Init` 使用 `sync.Once`；CLI DB 只走 `WithContext` / `FromContext`。

## 相关

- [架构与依赖注入](architecture.md)
- [Realtime 鉴权](../features/realtime.md)
- [Tracing / Metrics](../tracing.md)
