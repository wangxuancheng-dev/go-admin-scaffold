package v1

import (
	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

// ProfileHandler handles current-user profile endpoints.
type ProfileHandler struct {
	users services.UserServiceAPI
	rbac  *services.RBACService
}

func NewProfileHandler(users services.UserServiceAPI, rbac *services.RBACService) *ProfileHandler {
	return &ProfileHandler{users: users, rbac: rbac}
}

func (h *ProfileHandler) GetCurrentUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		response.UnauthorizedError(c)
		return
	}
	userModel := user.(*models.User)

	fullUser, err := h.users.GetByID(c.Request.Context(), userModel.ID)
	if err != nil {
		response.NotFoundError(c)
		return
	}
	permissions, err := h.rbac.GetUserPermissions(c.Request.Context(), userModel.ID)
	if err != nil {
		response.ServerError(c)
		return
	}
	response.Success(c, map[string]interface{}{
		"user":        fullUser,
		"permissions": permissions,
	})
}

func (h *ProfileHandler) UpdateCurrentUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		response.UnauthorizedError(c)
		return
	}
	userModel := user.(*models.User)

	var req struct {
		Email    string `json:"email"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	updatedUser, err := h.users.Update(c.Request.Context(), userModel.ID, &services.UpdateUserRequest{
		Email:    req.Email,
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
	})
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to update profile")
		return
	}
	response.Success(c, updatedUser)
}
