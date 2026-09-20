package v1

import (
	"strconv"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RoleHandler handles role management endpoints.
type RoleHandler struct {
	roles *services.RoleService
	menus *services.MenuService
}

func NewRoleHandler(roles *services.RoleService, menus *services.MenuService) *RoleHandler {
	return &RoleHandler{roles: roles, menus: menus}
}

func (h *RoleHandler) ListRoles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	pagination := &models.Pagination{Page: page, PageSize: pageSize}
	roles, err := h.roles.List(c.Request.Context(), pagination)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to fetch roles")
		return
	}
	response.PageSuccess(c, roles, pagination.Total, pagination.Page, pagination.PageSize)
}

func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req services.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	role, err := h.roles.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to create role")
		return
	}
	response.Success(c, role)
}

func (h *RoleHandler) GetRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid role ID")
		return
	}
	role, err := h.roles.GetByID(c.Request.Context(), uint(id))
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

func (h *RoleHandler) UpdateRole(c *gin.Context) {
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
	role, err := h.roles.Update(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to update role")
		return
	}
	response.Success(c, role)
}

func (h *RoleHandler) DeleteRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid role ID")
		return
	}
	if err := h.roles.Delete(c.Request.Context(), uint(id)); err != nil {
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

func (h *RoleHandler) GetRoleMenus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ParamError(c, "invalid role ID")
		return
	}
	allMenus, err := h.menus.GetAll(c.Request.Context())
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to get menus")
		return
	}
	roleMenus, err := h.roles.GetMenus(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to get role menus")
		return
	}
	assignedMap := make(map[uint]bool)
	for _, menu := range roleMenus {
		assignedMap[menu.ID] = true
	}
	response.Success(c, RoleMenuResponse{MenuTree: buildMenuTree(allMenus, assignedMap, nil)})
}

func (h *RoleHandler) UpdateRoleMenus(c *gin.Context) {
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
	if err := h.roles.UpdateMenus(c.Request.Context(), uint(id), &req); err != nil {
		if err == gorm.ErrRecordNotFound {
			response.NotFoundError(c)
			return
		}
		response.Error(c, response.CodeServerError, "failed to update role menus: "+err.Error())
		return
	}
	response.Success(c, nil)
}

func buildMenuTree(allMenus []models.Menu, assignedMap map[uint]bool, parentID *uint) []MenuTreeNode {
	var nodes []MenuTreeNode
	for _, menu := range allMenus {
		if menu.Status != 1 {
			continue
		}
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
