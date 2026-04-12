package v1

import (
	"strconv"

	"app/internal/core/models"
	"app/internal/core/services"
	"app/pkg/ginext"
	"app/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListRoles handles the request to list roles
func ListRoles(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	pagination := &models.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	roleSvc, ok := ginext.GetService[*services.RoleService](c, "roleService")
	if !ok {
		return
	}
	roles, err := roleSvc.List(c.Request.Context(), pagination)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to fetch roles")
		return
	}

	response.PageSuccess(c, roles, pagination.Total, pagination.Page, pagination.PageSize)
}

// CreateRole handles the request to create a new role
func CreateRole(c *gin.Context) {
	var req services.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	roleSvc, ok := ginext.GetService[*services.RoleService](c, "roleService")
	if !ok {
		return
	}
	role, err := roleSvc.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to create role")
		return
	}

	response.Success(c, role)
}

// GetRole handles the request to get a role by ID
func GetRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid role ID")
		return
	}

	roleSvc, ok := ginext.GetService[*services.RoleService](c, "roleService")
	if !ok {
		return
	}
	role, err := roleSvc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.NotFoundError(c)
			return
		}
		response.Error(c, response.CodeServerError, "failed to fetch role")
		return
	}

	response.Success(c, role)
}

// UpdateRole handles the request to update a role
func UpdateRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid role ID")
		return
	}

	var req services.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	roleSvc, ok := ginext.GetService[*services.RoleService](c, "roleService")
	if !ok {
		return
	}
	role, err := roleSvc.Update(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to update role")
		return
	}

	response.Success(c, role)
}

// DeleteRole handles the request to delete a role
func DeleteRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid role ID")
		return
	}

	roleSvc, ok := ginext.GetService[*services.RoleService](c, "roleService")
	if !ok {
		return
	}
	if err := roleSvc.Delete(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, response.CodeServerError, "failed to delete role")
		return
	}

	response.Success(c, nil)
}

// RoleMenuResponse represents the response for role menus
type RoleMenuResponse struct {
	MenuTree []MenuTreeNode `json:"menu_tree"`
}

// MenuTreeNode represents a node in the menu tree
type MenuTreeNode struct {
	ID       uint           `json:"id"`
	MenuID   uint           `json:"menu_id"`
	Name     string         `json:"name"`
	Title    string         `json:"title"`
	Icon     string         `json:"icon"`
	Path     string         `json:"path"`
	Status   int            `json:"status"`
	Assigned bool           `json:"assigned"`
	Children []MenuTreeNode `json:"children,omitempty"`
}

// GetRoleMenus handles the request to get menus of a role
func GetRoleMenus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid role ID")
		return
	}

	menuSvc, okMenu := ginext.GetService[*services.MenuService](c, "menuService")
	if !okMenu {
		return
	}
	roleSvc, okRole := ginext.GetService[*services.RoleService](c, "roleService")
	if !okRole {
		return
	}

	// Get all menus
	allMenus, err := menuSvc.GetAll(c.Request.Context())
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to get menus")
		return
	}

	// Get role's menus
	roleMenus, err := roleSvc.GetMenus(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to get role menus")
		return
	}

	// Create a map for quick lookup of assigned menus
	assignedMap := make(map[uint]bool)
	for _, menu := range roleMenus {
		assignedMap[menu.ID] = true
	}

	// Build menu tree
	var result RoleMenuResponse
	result.MenuTree = buildMenuTree(allMenus, assignedMap, nil)

	response.Success(c, result)
}

// UpdateRoleMenus handles the request to update menus of a role
func UpdateRoleMenus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid role ID")
		return
	}

	var req services.UpdateRoleMenusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	roleSvc, ok := ginext.GetService[*services.RoleService](c, "roleService")
	if !ok {
		return
	}
	if err := roleSvc.UpdateMenus(c.Request.Context(), uint(id), &req); err != nil {
		if err == gorm.ErrRecordNotFound {
			response.NotFoundError(c)
			return
		}
		response.Error(c, response.CodeServerError, "failed to update role menus: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// Helper function to build menu tree
func buildMenuTree(allMenus []models.Menu, assignedMap map[uint]bool, parentID *uint) []MenuTreeNode {
	var nodes []MenuTreeNode

	for _, menu := range allMenus {
		// Skip disabled menus
		if menu.Status != 1 {
			continue
		}

		// Check if this menu belongs to the current parent level
		if (parentID == nil && menu.ParentID == nil) || (parentID != nil && menu.ParentID != nil && *menu.ParentID == *parentID) {
			node := MenuTreeNode{
				ID:       menu.ID,
				MenuID:   menu.ID,
				Name:     menu.Name,
				Title:    menu.Title,
				Icon:     menu.Icon,
				Path:     menu.Path,
				Status:   menu.Status,
				Assigned: assignedMap[menu.ID],
				Children: buildMenuTree(allMenus, assignedMap, &menu.ID),
			}
			nodes = append(nodes, node)
		}
	}

	return nodes
}
