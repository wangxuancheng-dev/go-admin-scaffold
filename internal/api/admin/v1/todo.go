package v1

import (
	"strconv"

	"app/internal/core/models"
	"app/internal/core/services"
	"app/pkg/ginext"
	"app/pkg/response"

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

	todoSvc, ok := ginext.GetService[*services.TodoService](c, "todoService")
	if !ok {
		return
	}
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
// @Success 200 {object} response.Response{data=response.PagedList}
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

	todoSvc, ok := ginext.GetService[*services.TodoService](c, "todoService")
	if !ok {
		return
	}
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

	todoSvc, ok := ginext.GetService[*services.TodoService](c, "todoService")
	if !ok {
		return
	}
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

	todoSvc, ok := ginext.GetService[*services.TodoService](c, "todoService")
	if !ok {
		return
	}
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

	todoSvc, ok := ginext.GetService[*services.TodoService](c, "todoService")
	if !ok {
		return
	}
	if err := todoSvc.Delete(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, response.CodeServerError, "failed to delete todo")
		return
	}

	response.Success(c, nil)
}
