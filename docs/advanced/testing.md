# 测试指南

本文档详细说明了 Go Admin Scaffold 的测试策略和最佳实践。

## 测试类型

### 1. 单元测试

用于测试独立的代码单元（函数、方法、结构体等）。

```go
// internal/models/user_test.go
package models

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestUser_Validate(t *testing.T) {
    tests := []struct {
        name    string
        user    User
        wantErr bool
    }{
        {
            name: "valid user",
            user: User{
                Username: "testuser",
                Email:    "test@example.com",
                Password: "password123",
            },
            wantErr: false,
        },
        {
            name: "invalid email",
            user: User{
                Username: "testuser",
                Email:    "invalid-email",
                Password: "password123",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.user.Validate()
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 2. 集成测试

测试多个组件之间的交互。

```go
// internal/services/auth_test.go
package services

import (
    "testing"
    "github.com/1768177868/go-admin-scaffold/internal/models"
    "github.com/1768177868/go-admin-scaffold/pkg/database"
)

func TestAuthService_Login(t *testing.T) {
    // 设置测试数据库
    db, err := database.NewTestDB()
    assert.NoError(t, err)
    defer db.Close()

    // 创建测试用户
    user := &models.User{
        Username: "testuser",
        Email:    "test@example.com",
        Password: "password123",
    }
    err = db.Create(user).Error
    assert.NoError(t, err)

    // 测试登录
    authService := NewAuthService(db)
    token, err := authService.Login("test@example.com", "password123")
    assert.NoError(t, err)
    assert.NotEmpty(t, token)
}
```

### 3. API 测试

测试 HTTP API 端点。

```go
// internal/api/auth_test.go
package api

import (
    "testing"
    "net/http"
    "net/http/httptest"
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
)

func TestLoginHandler(t *testing.T) {
    // 设置测试路由
    router := gin.New()
    router.POST("/api/auth/login", LoginHandler)

    // 测试请求
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/api/auth/login", 
        strings.NewReader(`{"email":"test@example.com","password":"password123"}`))
    req.Header.Set("Content-Type", "application/json")
    router.ServeHTTP(w, req)

    // 验证响应
    assert.Equal(t, http.StatusOK, w.Code)
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.NotEmpty(t, response["token"])
}
```

### 4. 性能测试

测试代码性能和资源使用。

```go
// 示例：对 Asynq 驱动入队做基准测试（需可连 Redis）
package queue_test

import (
    "context"
    "testing"
    "time"

    "app/pkg/queue"
)

func BenchmarkAsynqQueue_Push(b *testing.B) {
    mgr, err := queue.NewManager(queue.Config{
        Driver: "redis",
        Options: map[string]interface{}{
            "connection": "redis://127.0.0.1:6379/15",
            "queue":      "bench",
        },
    })
    if err != nil {
        b.Fatal(err)
    }
    defer mgr.Close()

    job := &benchJob{BaseJob: queue.BaseJob{Queue: "bench", JobType: "bench", MaxAttempts: 1, Timeout: time.Minute}}
    ctx := context.Background()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = mgr.Push(ctx, job)
    }
}

type benchJob struct{ queue.BaseJob }

func (b *benchJob) Handle() error { return nil }
```

## 测试工具

### 1. 测试框架

- `testing` - Go 标准测试包
- `testify` - 断言和模拟工具
- `gomock` - 接口模拟
- `httptest` - HTTP 测试
- `go-sqlmock` - 数据库模拟

### 2. 测试覆盖率

```bash
# 运行测试并生成覆盖率报告
go test ./... -coverprofile=coverage.out

# 查看覆盖率报告
go tool cover -html=coverage.out

# 设置最低覆盖率要求
go test ./... -cover -covermode=atomic -coverpkg=./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total | awk '{print $3}' | cut -d. -f1
```

### 3. 测试数据

```go
// internal/database/fixtures/users.go
package fixtures

import "github.com/1768177868/go-admin-scaffold/internal/models"

var Users = []models.User{
    {
        Username: "admin",
        Email:    "admin@example.com",
        Password: "admin123",
        Role:     "admin",
    },
    {
        Username: "user",
        Email:    "user@example.com",
        Password: "user123",
        Role:     "user",
    },
}
```

## 测试最佳实践

### 1. 测试组织

- 测试文件与源文件放在同一目录
- 使用 `_test.go` 后缀
- 测试函数以 `Test` 开头
- 基准测试以 `Benchmark` 开头
- 示例测试以 `Example` 开头

### 2. 测试命名

```go
// 好的命名
func TestUser_Validate_WithValidData(t *testing.T)
func TestUser_Validate_WithInvalidEmail(t *testing.T)
func TestUser_Validate_WithEmptyPassword(t *testing.T)

// 基准测试命名
func BenchmarkUser_Validate(b *testing.B)
func BenchmarkUser_Save(b *testing.B)
```

### 3. 测试数据管理

- 使用测试夹具（fixtures）
- 使用工厂函数
- 使用测试数据库
- 清理测试数据

```go
// 工厂函数示例
func NewTestUser(t *testing.T) *models.User {
    user := &models.User{
        Username: "testuser",
        Email:    "test@example.com",
        Password: "password123",
    }
    err := user.Validate()
    assert.NoError(t, err)
    return user
}
```

### 4. 模拟和存根

```go
// 使用 gomock 模拟接口
type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) FindByID(id uint) (*models.User, error) {
    args := m.Called(id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*models.User), args.Error(1)
}

// 使用模拟对象
func TestUserService_GetUser(t *testing.T) {
    mockRepo := new(MockUserRepository)
    user := &models.User{ID: 1, Username: "testuser"}
    mockRepo.On("FindByID", uint(1)).Return(user, nil)

    service := NewUserService(mockRepo)
    result, err := service.GetUser(1)
    assert.NoError(t, err)
    assert.Equal(t, user, result)
}
```

## 测试环境

### 1. 开发环境

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/models

# 运行特定测试
go test ./internal/models -run TestUser_Validate

# 运行基准测试
go test ./pkg/queue -bench=.
```

### 2. CI/CD 环境

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - name: Run tests
        run: go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
      - name: Upload coverage
        uses: codecov/codecov-action@v2
```

## 常见问题

### 1. 测试失败

检查：
- 测试数据是否正确
- 环境变量是否设置
- 依赖服务是否可用
- 测试顺序是否影响

### 2. 测试性能

优化：
- 使用并行测试
- 减少数据库操作
- 使用内存数据库
- 优化测试数据

### 3. 测试维护

建议：
- 保持测试简单
- 避免测试间依赖
- 定期更新测试数据
- 及时修复失败的测试

## 相关文档

- [开发环境配置](development.md)
- [项目结构说明](../getting-started/structure.md)
- [API 文档](../api/README.md)
- [部署指南](../deployment/README.md) 