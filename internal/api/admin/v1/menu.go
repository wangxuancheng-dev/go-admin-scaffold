package v1

import (
	"strconv"

	"app/internal/core/models"
	"app/internal/core/services"
	"app/pkg/ginext"
	"app/pkg/logger"
	"app/pkg/response"

	"github.com/gin-gonic/gin"
)

// ListMenus handles the request to get all menus
// @Summary List menus
// @Description Get list of all menus
// @Tags menus
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]models.Menu}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/menus [get]
func ListMenus(c *gin.Context) {
	menuSvc, ok := ginext.GetService[*services.MenuService](c, "menuService")
	if !ok {
		return
	}
	menus, err := menuSvc.GetAll(c.Request.Context())
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to fetch menus")
		return
	}

	response.Success(c, menus)
}

// GetMenuTree handles the request to get menu tree
// @Summary Get menu tree
// @Description Get menu tree structure
// @Tags menus
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]models.Menu}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/menus/tree [get]
func GetMenuTree(c *gin.Context) {
	menuSvc, ok := ginext.GetService[*services.MenuService](c, "menuService")
	if !ok {
		return
	}
	tree, err := menuSvc.GetTree(c.Request.Context())
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to fetch menu tree")
		return
	}

	response.Success(c, tree)
}

// GetUserMenus handles the request to get user's accessible menus
// @Summary Get user menus
// @Description Get menus accessible by current user
// @Tags menus
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]services.MenuRouteItem}
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/menus/user [get]
func GetUserMenus(c *gin.Context) {
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

	menuSvc, ok := ginext.GetService[*services.MenuService](c, "menuService")
	if !ok {
		return
	}
	menus, err := menuSvc.GetUserMenus(ctx, userModel.ID)
	if err != nil {
		logger.Error(ctx, "GetUserMenus failed", "error", err, "user_id", userModel.ID)
		response.Error(c, response.CodeServerError, "failed to fetch user menus")
		return
	}

	response.Success(c, menus)
}

// CreateMenu handles the request to create a new menu
// @Summary Create menu
// @Description Create a new menu
// @Tags menus
// @Accept json
// @Produce json
// @Param menu body services.CreateMenuRequest true "Menu data"
// @Success 200 {object} response.Response{data=models.Menu}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/menus [post]
func CreateMenu(c *gin.Context) {
	var req services.CreateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	menuSvc, ok := ginext.GetService[*services.MenuService](c, "menuService")
	if !ok {
		return
	}
	menu, err := menuSvc.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to create menu")
		return
	}

	response.Success(c, menu)
}

// GetMenu handles the request to get a menu by ID
// @Summary Get menu
// @Description Get menu by ID
// @Tags menus
// @Accept json
// @Produce json
// @Param id path int true "Menu ID"
// @Success 200 {object} response.Response{data=models.Menu}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/menus/{id} [get]
func GetMenu(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid menu ID")
		return
	}

	menuSvc, ok := ginext.GetService[*services.MenuService](c, "menuService")
	if !ok {
		return
	}
	menu, err := menuSvc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.NotFoundError(c)
		return
	}

	response.Success(c, menu)
}

// UpdateMenu handles the request to update a menu
// @Summary Update menu
// @Description Update menu by ID
// @Tags menus
// @Accept json
// @Produce json
// @Param id path int true "Menu ID"
// @Param menu body services.UpdateMenuRequest true "Menu data"
// @Success 200 {object} response.Response{data=models.Menu}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/menus/{id} [put]
func UpdateMenu(c *gin.Context) {
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

	menuSvc, ok := ginext.GetService[*services.MenuService](c, "menuService")
	if !ok {
		return
	}
	menu, err := menuSvc.Update(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to update menu")
		return
	}

	response.Success(c, menu)
}

// DeleteMenu handles the request to delete a menu
// @Summary Delete menu
// @Description Delete menu by ID
// @Tags menus
// @Accept json
// @Produce json
// @Param id path int true "Menu ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/menus/{id} [delete]
func DeleteMenu(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid menu ID")
		return
	}

	menuSvc, ok := ginext.GetService[*services.MenuService](c, "menuService")
	if !ok {
		return
	}
	if err := menuSvc.Delete(c.Request.Context(), uint(id)); err != nil {
		if err == services.ErrMenuHasChildren {
			response.BusinessError(c, "cannot delete menu with children")
			return
		}
		response.Error(c, response.CodeServerError, "failed to delete menu")
		return
	}

	response.Success(c, nil)
}

// UpdateMenuRoles handles the request to update menu roles
// @Summary Update menu roles
// @Description Update roles associated with a menu
// @Tags menus
// @Accept json
// @Produce json
// @Param id path int true "Menu ID"
// @Param roles body UpdateMenuRolesRequest true "Role IDs"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Security Bearer
// @Router /admin/v1/menus/{id}/roles [put]
func UpdateMenuRoles(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid menu ID")
		return
	}

	var req struct {
		RoleIDs []uint `json:"role_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	menuSvc, ok := ginext.GetService[*services.MenuService](c, "menuService")
	if !ok {
		return
	}
	if err := menuSvc.UpdateMenuRoles(c.Request.Context(), uint(id), req.RoleIDs); err != nil {
		if err == services.ErrMenuNotFound {
			response.NotFoundError(c)
			return
		}
		response.Error(c, response.CodeServerError, "failed to update menu roles")
		return
	}

	response.Success(c, gin.H{"message": "Menu roles updated successfully"})
}

// UpdateMenuRolesRequest represents the request to update menu roles
type UpdateMenuRolesRequest struct {
	RoleIDs []uint `json:"role_ids" binding:"required"`
}
