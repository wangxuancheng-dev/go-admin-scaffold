package v1

import (
	"strconv"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/internal/core/types"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user management endpoints.
type UserHandler struct {
	users services.UserServiceAPI
	logs  *services.LogService
}

func NewUserHandler(users services.UserServiceAPI, logs *services.LogService) *UserHandler {
	return &UserHandler{users: users, logs: logs}
}

// ListUsers handles the request to list users with pagination and search
// @Summary List users
// @Description Get paginated list of users with optional search filters
// @Tags users
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Param username query string false "Username filter"
// @Param email query string false "Email filter"
// @Param status query int false "Status filter (0=inactive, 1=active)"
// @Param role_id query int false "Role ID filter"
// @Success 200 {object} response.Response{data=response.PagedList}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	filters := &types.UserSearchFilters{
		Username: c.Query("username"),
		Email:    c.Query("email"),
		RoleID:   0,
	}
	if statusStr := c.Query("status"); statusStr != "" {
		if status, err := strconv.Atoi(statusStr); err == nil {
			filters.Status = &status
		}
	}
	if roleIDStr := c.Query("role_id"); roleIDStr != "" {
		if roleID, err := strconv.ParseUint(roleIDStr, 10, 32); err == nil {
			filters.RoleID = uint(roleID)
		}
	}

	pagination := &models.Pagination{Page: page, PageSize: pageSize}
	users, err := h.users.ListWithFilters(c.Request.Context(), pagination, filters)
	if err != nil {
		response.ServerError(c)
		return
	}
	for i := range users {
		users[i].IsSuperAdmin = h.users.IsSuperAdmin(users[i].ID)
	}
	response.PageSuccess(c, users, pagination.Total, pagination.Page, pagination.PageSize)
}

// CreateUser handles the request to create a new user
// @Summary Create user
// @Description Create a new user
// @Tags users
// @Accept json
// @Produce json
// @Param user body services.CreateUserRequest true "User info"
// @Success 200 {object} response.Response{data=models.User}
// @Security Bearer
// @Router /admin/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req services.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	user, err := h.users.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to create user")
		return
	}
	response.Success(c, user)
}

// GetUser handles the request to get a user by ID
// @Summary Get user
// @Router /admin/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid user ID")
		return
	}
	user, err := h.users.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.NotFoundError(c)
		return
	}
	response.Success(c, user)
}

// UpdateUser handles the request to update a user
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid user ID")
		return
	}
	var req services.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	user, err := h.users.Update(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to update user")
		return
	}
	response.Success(c, user)
}

// DeleteUser handles the request to delete a user
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid user ID")
		return
	}
	if err := h.users.Delete(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, response.CodeServerError, "failed to delete user")
		return
	}
	response.Success(c, nil)
}

// ExportUsers handles the request to export user list data
func (h *UserHandler) ExportUsers(c *gin.Context) {
	var req services.ExportUserListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	users, err := h.users.ExportUserList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to export users")
		return
	}
	response.Success(c, users)
}

// UpdateUserRoles handles updating a user's roles
func (h *UserHandler) UpdateUserRoles(c *gin.Context) {
	var req struct {
		RoleIDs []uint `json:"role_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "Invalid user ID")
		return
	}
	if err := h.users.UpdateUserRoles(c.Request.Context(), uint(userID), req.RoleIDs); err != nil {
		response.BusinessError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "User roles updated successfully"})
}

// UpdateUserStatus handles the request to update a user's status
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid user ID")
		return
	}
	var req struct {
		Status *int `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	if err := h.users.UpdateStatus(c.Request.Context(), uint(id), *req.Status); err != nil {
		if err == services.ErrSuperAdminModify {
			response.BusinessError(c, "超级管理员账户状态不能修改")
			return
		}
		response.Error(c, response.CodeServerError, "failed to update user status")
		return
	}
	response.Success(c, gin.H{"message": "User status updated successfully"})
}

// GetUserLogs returns a user's login and operation logs
func (h *UserHandler) GetUserLogs(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "Invalid user ID")
		return
	}
	loginLogs, err := h.logs.GetUserLoginLogs(c.Request.Context(), uint(userID))
	if err != nil {
		response.ServerError(c)
		return
	}
	operationLogs, err := h.logs.GetUserOperationLogs(c.Request.Context(), uint(userID))
	if err != nil {
		response.ServerError(c)
		return
	}
	response.Success(c, gin.H{
		"login_logs":     loginLogs,
		"operation_logs": operationLogs,
	})
}
