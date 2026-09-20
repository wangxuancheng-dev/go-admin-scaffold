# Todo 应用示例

本教程将指导您使用我们的管理框架创建一个简单的待办事项(Todo)应用，展示基本的增删改查(CRUD)操作。

## 项目结构

```
.
├── internal
│   ├── core
│   │   ├── models
│   │   │   └── todo.go           # Todo 数据模型
│   │   ├── repositories
│   │   │   └── todo_repository.go # Todo 数据访问层
│   │   └── services
│   │       └── todo_service.go   # Todo 业务逻辑
│   ├── api
│   │   └── admin
│   │       └── v1
│   │           └── todo.go       # Todo 处理器和路由
│   └── database
│       └── migrations
│           └── create_todos_table.go  # 数据库迁移
```

## 1. 创建数据模型

创建 Todo 模型 (`internal/core/models/todo.go`):

```go
package models

import (
    "go-admin-scaffold/internal/core/models"
)

type Todo struct {
    models.BaseModel
    Title       string `json:"title" gorm:"not null"`
    Description string `json:"description"`
    Completed   bool   `json:"completed" gorm:"default:false"`
}

func (Todo) TableName() string {
    return "todos"
}
```

## 2. 创建数据库迁移

### 2.1 生成迁移文件

使用命令生成迁移文件（类似 Laravel）：

```bash
# 基本迁移
go run cmd/tools/main.go make:migration create_todos_table

# 创建表的迁移（带模板）
go run cmd/tools/main.go make:migration create_todos_table --create=todos

# 修改表的迁移（带模板）
go run cmd/tools/main.go make:migration add_status_to_todos --table=todos
```


### 2.2 编辑迁移文件

生成的迁移文件位于 `internal/database/migrations/` 目录下。对于 Todo 示例，编辑生成的文件：

```go
package migrations

import (
	"time"
	"go-admin-scaffold/internal/core/models"
	"gorm.io/gorm"
)

func init() {
	Register("create_todos_table", &MigrationDefinition{
		Up: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&models.Todo{})
		},
		Down: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable("todos")
		},
	})
}
```

### 2.3 运行迁移

```bash
# 运行所有待执行的迁移
go run cmd/tools/main.go migrate run

# 查看迁移状态
go run cmd/tools/main.go migrate status

# 回滚最后一批迁移
go run cmd/tools/main.go migrate rollback

# 重置所有迁移
go run cmd/tools/main.go migrate reset

# 重置并重新运行所有迁移
go run cmd/tools/main.go migrate refresh
```

### 2.4 迁移命令选项

- `--create=table_name`: 生成创建表的迁移模板
- `--table=table_name`: 生成修改表的迁移模板

示例：
```bash
# 创建用户表
go run cmd/tools/main.go make:migration create_users_table --create=users

# 为用户表添加字段
go run cmd/tools/main.go make:migration add_phone_to_users --table=users

# 创建索引
go run cmd/tools/main.go make:migration add_index_to_users_email
```

## 3. 创建数据访问层

创建 Todo 仓库 (`internal/core/repositories/todo_repository.go`):

```go
package repositories

import (
    "context"
    "go-admin-scaffold/internal/core/models"
    "gorm.io/gorm"
)

type TodoRepository struct {
    db *gorm.DB
}

func NewTodoRepository(db *gorm.DB) *TodoRepository {
    return &TodoRepository{db: db}
}

func (r *TodoRepository) Create(ctx context.Context, todo *models.Todo) error {
    return r.db.Create(todo).Error
}

func (r *TodoRepository) List(ctx context.Context, pagination *models.Pagination) ([]models.Todo, error) {
    var todos []models.Todo
    query := r.db.Model(&models.Todo{})
    
    if err := query.Count(&pagination.Total).Error; err != nil {
        return nil, err
    }
    
    if err := query.Offset(pagination.GetOffset()).
        Limit(pagination.GetLimit()).
        Order("created_at DESC").
        Find(&todos).Error; err != nil {
        return nil, err
    }
    
    return todos, nil
}

func (r *TodoRepository) GetByID(ctx context.Context, id uint) (*models.Todo, error) {
    var todo models.Todo
    if err := r.db.First(&todo, id).Error; err != nil {
        return nil, err
    }
    return &todo, nil
}

func (r *TodoRepository) Update(ctx context.Context, todo *models.Todo) error {
    return r.db.Save(todo).Error
}

func (r *TodoRepository) Delete(ctx context.Context, id uint) error {
    return r.db.Delete(&models.Todo{}, id).Error
}
```

## 4. 创建服务层

创建 Todo 服务 (`internal/core/services/todo_service.go`):

```go
package services

import (
    "context"
    "go-admin-scaffold/internal/core/models"
    "go-admin-scaffold/internal/core/repositories"
)

type TodoService struct {
    repo *repositories.TodoRepository
}

func NewTodoService(repo *repositories.TodoRepository) *TodoService {
    return &TodoService{repo: repo}
}

type CreateTodoRequest struct {
    Title       string `json:"title" binding:"required"`
    Description string `json:"description"`
}

type UpdateTodoRequest struct {
    Title       string `json:"title" binding:"required"`
    Description string `json:"description"`
    Completed   bool   `json:"completed"`
}

func (s *TodoService) Create(ctx context.Context, req *CreateTodoRequest) (*models.Todo, error) {
    todo := &models.Todo{
        Title:       req.Title,
        Description: req.Description,
    }
    if err := s.repo.Create(ctx, todo); err != nil {
        return nil, err
    }
    return todo, nil
}

func (s *TodoService) List(ctx context.Context, pagination *models.Pagination) ([]models.Todo, error) {
    return s.repo.List(ctx, pagination)
}

func (s *TodoService) GetByID(ctx context.Context, id uint) (*models.Todo, error) {
    return s.repo.GetByID(ctx, id)
}

func (s *TodoService) Update(ctx context.Context, id uint, req *UpdateTodoRequest) (*models.Todo, error) {
    todo, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    todo.Title = req.Title
    todo.Description = req.Description
    todo.Completed = req.Completed
    
    if err := s.repo.Update(ctx, todo); err != nil {
        return nil, err
    }
    return todo, nil
}

func (s *TodoService) Delete(ctx context.Context, id uint) error {
    return s.repo.Delete(ctx, id)
}
```

## 5. 创建处理器

创建 Todo 处理器 (`internal/api/admin/v1/todo.go`):

```go
package v1

import (
	"strconv"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

// CreateTodo handles the request to create a new todo item
// @Summary Create todo
// @Description Create a new todo item
// @Tags todos
// @Accept json
// @Produce json
// @Param todo body services.CreateTodoRequest true "Todo info"
// @Success 200 {object} response.Response{data=models.Todo}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/todos [post]
func CreateTodo(c *gin.Context) {
	var req services.CreateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	todoSvc := c.MustGet("todoService").(*services.TodoService)
	todo, err := todoSvc.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to create todo")
		return
	}

	response.Success(c, todo)
}

// ListTodos handles the request to get a paginated list of todos
// @Summary List todos
// @Description Get a paginated list of todos
// @Tags todos
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} response.Response{data=response.PageData{list=[]models.Todo}}
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/todos [get]
func ListTodos(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	pagination := &models.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	todoSvc := c.MustGet("todoService").(*services.TodoService)
	todos, err := todoSvc.List(c.Request.Context(), pagination)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to fetch todos")
		return
	}

	response.PageSuccess(c, todos, pagination.Total, pagination.Page, pagination.PageSize)
}

// GetTodo handles the request to get a todo by ID
// @Summary Get todo
// @Description Get todo by ID
// @Tags todos
// @Accept json
// @Produce json
// @Param id path int true "Todo ID"
// @Success 200 {object} response.Response{data=models.Todo}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/todos/{id} [get]
func GetTodo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid todo ID")
		return
	}

	todoSvc := c.MustGet("todoService").(*services.TodoService)
	todo, err := todoSvc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.NotFoundError(c)
		return
	}

	response.Success(c, todo)
}

// UpdateTodo handles the request to update a todo
// @Summary Update todo
// @Description Update todo by ID
// @Tags todos
// @Accept json
// @Produce json
// @Param id path int true "Todo ID"
// @Param todo body services.UpdateTodoRequest true "Todo info"
// @Success 200 {object} response.Response{data=models.Todo}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/todos/{id} [put]
func UpdateTodo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid todo ID")
		return
	}

	var req services.UpdateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	todoSvc := c.MustGet("todoService").(*services.TodoService)
	todo, err := todoSvc.Update(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to update todo")
		return
	}

	response.Success(c, todo)
}

// DeleteTodo handles the request to delete a todo
// @Summary Delete todo
// @Description Delete todo by ID
// @Tags todos
// @Accept json
// @Produce json
// @Param id path int true "Todo ID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/todos/{id} [delete]
func DeleteTodo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid todo ID")
		return
	}

	todoSvc := c.MustGet("todoService").(*services.TodoService)
	if err := todoSvc.Delete(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, response.CodeServerError, "failed to delete todo")
		return
	}

	response.Success(c, nil)
}
```

## 6. 注册路由

在 `internal/routes/router.go` 中的 `adminV1Protected` 路由组中添加 Todo 路由：

```go
// Todo routes
todos := adminV1Protected.Group("/todos")
{
	todos.GET("", rbac("todo:view"), wrapHandler(api.Todos.ListTodos))
	todos.POST("", rbac("todo:create"), wrapHandler(api.Todos.CreateTodo))
	todos.GET("/:id", rbac("todo:view"), wrapHandler(api.Todos.GetTodo))
	todos.PUT("/:id", rbac("todo:edit"), wrapHandler(api.Todos.UpdateTodo))
	todos.DELETE("/:id", rbac("todo:delete"), wrapHandler(api.Todos.DeleteTodo))
}
```

## 7. 注册服务

在组合根 `internal/bootstrap/container.go` 中组装 Todo（脚手架已默认包含）：

```go
todoRepo := repositories.NewTodoRepository(db)
todoSvc := services.NewTodoService(todoRepo)
// 放入 Container，再由 adminv1.NewAdminAPI(c) 构造 TodoHandler
```

路由侧使用构造注入的 handler，例如 `api.Todos.ListTodos`。

## 8. API 使用示例

### 创建待办事项
```bash
curl -X POST http://localhost:8080/api/admin/v1/todos \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "title": "完成项目文档",
    "description": "编写项目的技术文档和使用说明"
  }'
```

### 获取待办事项列表
```bash
curl "http://localhost:8080/api/admin/v1/todos?page=1&page_size=10" \
  -H "Authorization: Bearer your-token"
```

### 获取单个待办事项
```bash
curl "http://localhost:8080/api/admin/v1/todos/1" \
  -H "Authorization: Bearer your-token"
```

### 更新待办事项
```bash
curl -X PUT http://localhost:8080/api/admin/v1/todos/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "title": "完成项目文档",
    "description": "编写项目的技术文档和使用说明",
    "completed": true
  }'
```

### 删除待办事项
```bash
curl -X DELETE http://localhost:8080/api/admin/v1/todos/1 \
  -H "Authorization: Bearer your-token"
```

## 9. 运行项目

1. 运行数据库迁移：
```bash
go run cmd/tools/main.go migrate run
```

2. 启动服务器：
```bash
go run cmd/server/main.go
```

3. 访问 Swagger 文档：
```
http://localhost:8080/swagger/index.html
```

## 总结

这个示例展示了如何使用我们的框架构建一个完整的 Todo CRUD 应用，包括：

1. 分层架构：
   - 模型层 (Models)
   - 数据访问层 (Repositories)
   - 服务层 (Services)
   - 处理器层 (Handlers)
   - API 路由层 (Routes)
2. 数据库迁移
3. 权限控制
4. API 文档
5. 统一的响应格式
6. 错误处理

通过这个示例，您可以了解到框架的主要特性和最佳实践。每一层都有其特定的职责：
- Models: 定义数据结构和验证规则
- Repositories: 处理数据持久化，封装数据库操作
- Services: 实现业务逻辑，协调多个仓库
- Handlers: 处理 HTTP 请求，参数验证，调用服务层
- Routes: 定义 API 路由，配置中间件

### 关键要点

1. **服务注入**：通过 `c.MustGet("serviceName")` 获取服务实例
2. **参数解析**：使用 `strconv.ParseUint` 解析路径参数
3. **错误处理**：使用统一的错误响应格式
4. **权限控制**：通过 RBAC 中间件控制访问权限
5. **API 文档**：使用 Swagger 注解自动生成文档