package v1

import (
	"strconv"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/pkg/logger"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

// MenuHandler handles menu management endpoints.
type MenuHandler struct {
	menus *services.MenuService
}

func NewMenuHandler(menus *services.MenuService) *MenuHandler {
	return &MenuHandler{menus: menus}
}

func (h *MenuHandler) ListMenus(c *gin.Context) {
	menus, err := h.menus.GetAll(c.Request.Context())
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to fetch menus")
		return
	}
	response.Success(c, menus)
}

func (h *MenuHandler) GetMenuTree(c *gin.Context) {
	tree, err := h.menus.GetTree(c.Request.Context())
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to fetch menu tree")
		return
	}
	response.Success(c, tree)
}

func (h *MenuHandler) GetUserMenus(c *gin.Context) {
	ctx := c.Request.Context()
	user, exists := c.Get("user")
	if !exists {
		response.UnauthorizedError(c)
		return
	}
	userModel, ok := user.(*models.User)
	if !ok {
		logger.Warn(ctx, "GetUserMenus: invalid user type in context")
		response.UnauthorizedError(c)
		return
	}
	menus, err := h.menus.GetUserMenus(ctx, userModel.ID)
	if err != nil {
		logger.Error(ctx, "GetUserMenus failed", "error", err, "user_id", userModel.ID)
		response.Error(c, response.CodeServerError, "failed to fetch user menus")
		return
	}
	response.Success(c, menus)
}

func (h *MenuHandler) CreateMenu(c *gin.Context) {
	var req services.CreateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	menu, err := h.menus.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to create menu")
		return
	}
	response.Success(c, menu)
}

func (h *MenuHandler) GetMenu(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid menu ID")
		return
	}
	menu, err := h.menus.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.NotFoundError(c)
		return
	}
	response.Success(c, menu)
}

func (h *MenuHandler) UpdateMenu(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid menu ID")
		return
	}
	var req services.UpdateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	menu, err := h.menus.Update(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to update menu")
		return
	}
	response.Success(c, menu)
}

func (h *MenuHandler) DeleteMenu(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid menu ID")
		return
	}
	if err := h.menus.Delete(c.Request.Context(), uint(id)); err != nil {
		if err == services.ErrMenuHasChildren {
			response.BusinessError(c, "cannot delete menu with children")
			return
		}
		response.Error(c, response.CodeServerError, "failed to delete menu")
		return
	}
	response.Success(c, nil)
}

// UpdateMenuRolesRequest represents the request to update menu roles
type UpdateMenuRolesRequest struct {
	RoleIDs []uint `json:"role_ids" binding:"required"`
}

func (h *MenuHandler) UpdateMenuRoles(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid menu ID")
		return
	}
	var req UpdateMenuRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	if err := h.menus.UpdateMenuRoles(c.Request.Context(), uint(id), req.RoleIDs); err != nil {
		if err == services.ErrMenuNotFound {
			response.NotFoundError(c)
			return
		}
		response.Error(c, response.CodeServerError, "failed to update menu roles")
		return
	}
	response.Success(c, gin.H{"message": "Menu roles updated successfully"})
}
