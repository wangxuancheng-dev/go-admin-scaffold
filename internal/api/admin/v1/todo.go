package v1

import (
	"strconv"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

// TodoHandler handles todo CRUD endpoints.
type TodoHandler struct {
	todos *services.TodoService
}

func NewTodoHandler(todos *services.TodoService) *TodoHandler {
	return &TodoHandler{todos: todos}
}

func (h *TodoHandler) CreateTodo(c *gin.Context) {
	var req services.CreateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	todo, err := h.todos.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to create todo")
		return
	}
	response.Success(c, todo)
}

func (h *TodoHandler) ListTodos(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	pagination := &models.Pagination{Page: page, PageSize: pageSize}
	todos, err := h.todos.List(c.Request.Context(), pagination)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to fetch todos")
		return
	}
	response.PageSuccess(c, todos, pagination.Total, pagination.Page, pagination.PageSize)
}

func (h *TodoHandler) GetTodo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid todo ID")
		return
	}
	todo, err := h.todos.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.NotFoundError(c)
		return
	}
	response.Success(c, todo)
}

func (h *TodoHandler) UpdateTodo(c *gin.Context) {
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
	todo, err := h.todos.Update(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to update todo")
		return
	}
	response.Success(c, todo)
}

func (h *TodoHandler) DeleteTodo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid todo ID")
		return
	}
	if err := h.todos.Delete(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, response.CodeServerError, "failed to delete todo")
		return
	}
	response.Success(c, nil)
}
