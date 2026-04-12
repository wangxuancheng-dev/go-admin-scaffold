package v1

import (
	"strconv"

	"app/internal/core/models"
	"app/internal/core/services"
	"app/internal/core/types"
	"app/pkg/ginext"
	"app/pkg/response"

	"github.com/gin-gonic/gin"
)

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
func ListUsers(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// Parse search filters
	filters := &types.UserSearchFilters{
		Username: c.Query("username"),
		Email:    c.Query("email"),
		RoleID:   0,
	}

	// Parse status filter
	if statusStr := c.Query("status"); statusStr != "" {
		if status, err := strconv.Atoi(statusStr); err == nil {
			filters.Status = &status
		}
	}

	// Parse role_id filter
	if roleIDStr := c.Query("role_id"); roleIDStr != "" {
		if roleID, err := strconv.ParseUint(roleIDStr, 10, 32); err == nil {
			filters.RoleID = uint(roleID)
		}
	}

	pagination := &models.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	userSvc, ok := ginext.GetService[services.UserServiceAPI](c, "userService")
	if !ok {
		return
	}
	users, err := userSvc.ListWithFilters(c.Request.Context(), pagination, filters)
	if err != nil {
		response.ServerError(c)
		return
	}

	// 确保每个用户的 is_super_admin 字段被正确设置
	for i := range users {
		users[i].IsSuperAdmin = userSvc.IsSuperAdmin(users[i].ID)
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
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/users [post]
func CreateUser(c *gin.Context) {
	var req services.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	userSvc, ok := ginext.GetService[services.UserServiceAPI](c, "userService")
	if !ok {
		return
	}
	user, err := userSvc.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to create user")
		return
	}

	response.Success(c, user)
}

// GetUser handles the request to get a user by ID
// @Summary Get user
// @Description Get user by ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} response.Response{data=models.User}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/users/{id} [get]
func GetUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid user ID")
		return
	}

	userSvc, ok := ginext.GetService[services.UserServiceAPI](c, "userService")
	if !ok {
		return
	}
	user, err := userSvc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.NotFoundError(c)
		return
	}

	response.Success(c, user)
}

// UpdateUser handles the request to update a user
func UpdateUser(c *gin.Context) {
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

	userSvc, ok := ginext.GetService[services.UserServiceAPI](c, "userService")
	if !ok {
		return
	}
	user, err := userSvc.Update(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to update user")
		return
	}

	response.Success(c, user)
}

// DeleteUser handles the request to delete a user
func DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid user ID")
		return
	}

	userSvc, ok := ginext.GetService[services.UserServiceAPI](c, "userService")
	if !ok {
		return
	}
	if err := userSvc.Delete(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, response.CodeServerError, "failed to delete user")
		return
	}

	response.Success(c, nil)
}

// ExportUsers handles the request to export user list data
func ExportUsers(c *gin.Context) {
	var req services.ExportUserListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	userSvc, ok := ginext.GetService[services.UserServiceAPI](c, "userService")
	if !ok {
		return
	}
	users, err := userSvc.ExportUserList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to export users")
		return
	}

	response.Success(c, users)
}

// UpdateUserRoles handles updating a user's roles
func UpdateUserRoles(c *gin.Context) {
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

	userSvc, ok := ginext.GetService[services.UserServiceAPI](c, "userService")
	if !ok {
		return
	}
	if err := userSvc.UpdateUserRoles(c.Request.Context(), uint(userID), req.RoleIDs); err != nil {
		response.BusinessError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "User roles updated successfully"})
}

// UpdateUserStatus handles the request to update a user's status
func UpdateUserStatus(c *gin.Context) {
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

	status := *req.Status

	userSvc, ok := ginext.GetService[services.UserServiceAPI](c, "userService")
	if !ok {
		return
	}
	if err := userSvc.UpdateStatus(c.Request.Context(), uint(id), status); err != nil {
		if err == services.ErrSuperAdminModify {
			response.BusinessError(c, "超级管理员账户状态不能修改")
			return
		}
		response.Error(c, response.CodeServerError, "failed to update user status")
		return
	}

	response.Success(c, gin.H{"message": "User status updated successfully"})
}
