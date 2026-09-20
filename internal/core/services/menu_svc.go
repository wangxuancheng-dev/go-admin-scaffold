package services

import (
	"context"
	"encoding/json"
	"errors"
	"sort"

	"go-admin-scaffold/internal/core/models"
)

var (
	ErrMenuNotFound    = errors.New("menu not found")
	ErrMenuHasChildren = errors.New("menu has children, cannot delete")
)

type MenuService struct {
	menuRepo MenuRepository
	userRepo UserRepository
	permInv  PermissionCacheInvalidator
}

func NewMenuService(menuRepo MenuRepository, userRepo UserRepository, permInv PermissionCacheInvalidator) *MenuService {
	return &MenuService{
		menuRepo: menuRepo,
		userRepo: userRepo,
		permInv:  permInv,
	}
}

func (s *MenuService) invalidateAllPermissions(ctx context.Context) {
	if s.permInv != nil {
		s.permInv.InvalidateAllPermissions(ctx)
	}
}

type CreateMenuRequest struct {
	Name       string          `json:"name" binding:"required"`
	Title      string          `json:"title" binding:"required"`
	Icon       string          `json:"icon"`
	Path       string          `json:"path"`
	Component  string          `json:"component"`
	ParentID   *uint           `json:"parent_id"`
	Sort       int             `json:"sort"`
	Type       int             `json:"type"`
	Visible    int             `json:"visible"`
	Status     int             `json:"status"`
	KeepAlive  bool            `json:"keep_alive"`
	External   bool            `json:"external"`
	Permission string          `json:"permission"`
	Meta       models.MenuMeta `json:"meta"`
	RoleIDs    []uint          `json:"role_ids"`
}

type UpdateMenuRequest struct {
	Name       string          `json:"name"`
	Title      string          `json:"title"`
	Icon       string          `json:"icon"`
	Path       string          `json:"path"`
	Component  string          `json:"component"`
	ParentID   *uint           `json:"parent_id"`
	Sort       int             `json:"sort"`
	Type       int             `json:"type"`
	Visible    int             `json:"visible"`
	Status     int             `json:"status"`
	KeepAlive  bool            `json:"keep_alive"`
	External   bool            `json:"external"`
	Permission string          `json:"permission"`
	Meta       models.MenuMeta `json:"meta"`
	RoleIDs    []uint          `json:"role_ids"`
}

// MenuMeta represents the metadata for a menu route
type MenuMeta struct {
	Title     string `json:"title"`
	Icon      string `json:"icon,omitempty"`
	KeepAlive bool   `json:"keepAlive,omitempty"`
	Hidden    bool   `json:"hidden,omitempty"`
}

// MenuRouteItem represents a menu item in the route tree
type MenuRouteItem struct {
	ID        uint            `json:"id"`
	Name      string          `json:"name"`
	Path      string          `json:"path"`
	Component string          `json:"component"`
	Meta      MenuMeta        `json:"meta"`
	Children  []MenuRouteItem `json:"children"`
	Sort      int             `json:"sort"`
}

// Create creates a new menu
func (s *MenuService) Create(ctx context.Context, req *CreateMenuRequest) (*models.Menu, error) {
	// Set sort value if not provided
	if req.Sort == 0 {
		maxSort, err := s.menuRepo.GetMaxSort(ctx, req.ParentID)
		if err != nil {
			return nil, err
		}
		req.Sort = maxSort + 1
	}

	menu := &models.Menu{
		Name:       req.Name,
		Title:      req.Title,
		Icon:       req.Icon,
		Path:       req.Path,
		Component:  req.Component,
		ParentID:   req.ParentID,
		Sort:       req.Sort,
		Type:       req.Type,
		Visible:    req.Visible,
		Status:     req.Status,
		KeepAlive:  req.KeepAlive,
		External:   req.External,
		Permission: req.Permission,
		Meta:       s.metaToString(req.Meta),
	}

	if err := s.menuRepo.Create(ctx, menu); err != nil {
		return nil, err
	}

	// Update role associations if provided
	if len(req.RoleIDs) > 0 {
		if err := s.menuRepo.UpdateMenuRoles(ctx, menu.ID, req.RoleIDs); err != nil {
			return nil, err
		}
	}

	s.invalidateAllPermissions(ctx)
	return s.menuRepo.FindByID(ctx, menu.ID)
}

// Update updates an existing menu
func (s *MenuService) Update(ctx context.Context, id uint, req *UpdateMenuRequest) (*models.Menu, error) {
	menu, err := s.menuRepo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrMenuNotFound
	}

	// Update fields
	if req.Name != "" {
		menu.Name = req.Name
	}
	if req.Title != "" {
		menu.Title = req.Title
	}
	menu.Icon = req.Icon
	menu.Path = req.Path
	menu.Component = req.Component
	menu.ParentID = req.ParentID
	if req.Sort != 0 {
		menu.Sort = req.Sort
	}
	if req.Type != 0 {
		menu.Type = req.Type
	}
	if req.Visible != 0 {
		menu.Visible = req.Visible
	}
	if req.Status != 0 {
		menu.Status = req.Status
	}
	menu.KeepAlive = req.KeepAlive
	menu.External = req.External
	menu.Permission = req.Permission
	menu.Meta = s.metaToString(req.Meta)

	if err := s.menuRepo.Update(ctx, menu); err != nil {
		return nil, err
	}

	// Update role associations if provided
	if req.RoleIDs != nil {
		if err := s.menuRepo.UpdateMenuRoles(ctx, menu.ID, req.RoleIDs); err != nil {
			return nil, err
		}
	}

	s.invalidateAllPermissions(ctx)
	return s.menuRepo.FindByID(ctx, menu.ID)
}

// Delete deletes a menu
func (s *MenuService) Delete(ctx context.Context, id uint) error {
	// Check if menu has children
	children, err := s.menuRepo.FindByParentID(ctx, &id)
	if err != nil {
		return err
	}
	if len(children) > 0 {
		return ErrMenuHasChildren
	}

	if err := s.menuRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidateAllPermissions(ctx)
	return nil
}

// GetByID gets a menu by ID
func (s *MenuService) GetByID(ctx context.Context, id uint) (*models.Menu, error) {
	return s.menuRepo.FindByID(ctx, id)
}

// GetTree gets the menu tree
func (s *MenuService) GetTree(ctx context.Context) ([]models.Menu, error) {
	return s.menuRepo.FindTree(ctx)
}

// GetAll gets all menus
func (s *MenuService) GetAll(ctx context.Context) ([]models.Menu, error) {
	return s.menuRepo.FindAll(ctx)
}

// GetUserMenus gets menus accessible by a user
func (s *MenuService) GetUserMenus(ctx context.Context, userID uint) ([]MenuRouteItem, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check if user is super admin - use the IsSuperAdmin field from user model
	if user.IsSuperAdmin {
		// Get all menus first, then filter visible and enabled ones
		allMenus, err := s.menuRepo.FindAll(ctx)
		if err != nil {
			return nil, err
		}

		// Filter for visible and enabled menus
		var menus []models.Menu
		for _, menu := range allMenus {
			if menu.Visible == 1 && menu.Status == 1 {
				menus = append(menus, menu)
			}
		}

		// If no visible menus found, create a default dashboard menu
		if len(menus) == 0 {
			defaultMenus := []MenuRouteItem{
				{
					ID:        1,
					Name:      "Dashboard",
					Path:      "/dashboard",
					Component: "@/views/dashboard/index.vue",
					Meta: MenuMeta{
						Title:     "仪表盘",
						Icon:      "Odometer",
						KeepAlive: false,
						Hidden:    false,
					},
					Children: []MenuRouteItem{},
					Sort:     0,
				},
			}
			return defaultMenus, nil
		}

		return s.buildMenuRoutes(menus), nil
	}

	// Extract role IDs for non-super-admin users
	roleIDs := make([]uint, 0)
	for _, role := range user.Roles {
		if role.Status == 1 { // Only include active roles
			roleIDs = append(roleIDs, role.ID)
		}
	}

	// If user has no active roles, return empty menu list
	if len(roleIDs) == 0 {
		return []MenuRouteItem{}, nil
	}

	// Get menus by role IDs
	menus, err := s.menuRepo.FindByRoleIDs(ctx, roleIDs)
	if err != nil {
		return nil, err
	}

	// Filter visible and enabled menus
	var visibleMenus []models.Menu
	for _, menu := range menus {
		if menu.Visible == 1 && menu.Status == 1 {
			visibleMenus = append(visibleMenus, menu)
		}
	}

	return s.buildMenuRoutes(visibleMenus), nil
}

// GetVisibleMenuTree gets the visible menu tree for public access
func (s *MenuService) GetVisibleMenuTree(ctx context.Context) ([]MenuRouteItem, error) {
	menus, err := s.menuRepo.FindVisibleMenus(ctx)
	if err != nil {
		return nil, err
	}

	// Filter menu type only
	var menuItems []models.Menu
	for _, menu := range menus {
		if menu.IsMenu() {
			menuItems = append(menuItems, menu)
		}
	}

	return s.buildMenuRoutes(menuItems), nil
}

// buildMenuRoutes builds menu route items from menu models
func (s *MenuService) buildMenuRoutes(menus []models.Menu) []MenuRouteItem {
	menuMap := make(map[uint]*MenuRouteItem)
	var rootMenus []*MenuRouteItem

	// First pass: create menu items
	for _, menu := range menus {
		// Parse meta JSON string to extract metadata
		var metaData MenuMeta
		if menu.Meta != "" {
			if err := json.Unmarshal([]byte(menu.Meta), &metaData); err != nil {
				// Use default meta values on parse failure
				metaData = MenuMeta{
					Title:     menu.Title,
					Icon:      menu.Icon,
					KeepAlive: menu.KeepAlive,
					Hidden:    menu.Visible != 1,
				}
			}
		} else {
			// Use default meta values if meta is empty
			metaData = MenuMeta{
				Title:     menu.Title,
				Icon:      menu.Icon,
				KeepAlive: menu.KeepAlive,
				Hidden:    menu.Visible != 1,
			}
		}

		item := &MenuRouteItem{
			ID:        menu.ID,
			Name:      menu.Name,
			Path:      menu.Path,
			Component: menu.Component,
			Meta:      metaData,
			Children:  make([]MenuRouteItem, 0),
			Sort:      menu.Sort,
		}
		menuMap[menu.ID] = item
	}

	// Second pass: build tree structure
	for _, menu := range menus {
		item := menuMap[menu.ID]

		if menu.ParentID == nil {
			rootMenus = append(rootMenus, item)
		} else {
			parentID := *menu.ParentID
			if parent, exists := menuMap[parentID]; exists {
				parent.Children = append(parent.Children, *item)
			}
		}
	}

	// Sort children by sort field
	for _, root := range rootMenus {
		sort.Slice(root.Children, func(i, j int) bool {
			return root.Children[i].Sort < root.Children[j].Sort
		})
	}

	// Sort root menus by sort field
	sort.Slice(rootMenus, func(i, j int) bool {
		return rootMenus[i].Sort < rootMenus[j].Sort
	})

	// Convert pointer slice to value slice for return
	result := make([]MenuRouteItem, len(rootMenus))
	for i, root := range rootMenus {
		result[i] = *root
	}

	return result
}

// UpdateMenuRoles updates the roles associated with a menu
func (s *MenuService) UpdateMenuRoles(ctx context.Context, menuID uint, roleIDs []uint) error {
	// Check if menu exists
	_, err := s.menuRepo.FindByID(ctx, menuID)
	if err != nil {
		return ErrMenuNotFound
	}

	if err := s.menuRepo.UpdateMenuRoles(ctx, menuID, roleIDs); err != nil {
		return err
	}
	s.invalidateAllPermissions(ctx)
	return nil
}

func (s *MenuService) metaToString(meta models.MenuMeta) string {
	jsonData, err := json.Marshal(meta)
	if err != nil {
		return ""
	}
	return string(jsonData)
}
