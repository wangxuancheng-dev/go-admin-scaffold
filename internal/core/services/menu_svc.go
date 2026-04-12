package services

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sort"

	"app/internal/core/models"
)

var (
	ErrMenuNotFound    = errors.New("menu not found")
	ErrMenuHasChildren = errors.New("menu has children, cannot delete")
)

type MenuRepository interface {
	FindByID(ctx context.Context, id uint) (*models.Menu, error)
	FindAll(ctx context.Context) ([]models.Menu, error)
	FindByParentID(ctx context.Context, parentID *uint) ([]models.Menu, error)
	FindTree(ctx context.Context) ([]models.Menu, error)
	FindByRoleIDs(ctx context.Context, roleIDs []uint) ([]models.Menu, error)
	FindVisibleMenus(ctx context.Context) ([]models.Menu, error)
	Create(ctx context.Context, menu *models.Menu) error
	Update(ctx context.Context, menu *models.Menu) error
	Delete(ctx context.Context, id uint) error
	UpdateMenuRoles(ctx context.Context, menuID uint, roleIDs []uint) error
	GetMaxSort(ctx context.Context, parentID *uint) (int, error)
}

type MenuService struct {
	menuRepo MenuRepository
	userRepo UserRepository
	permInv  PermissionCacheInvalidator
}

func NewMenuService(menuRepo MenuRepository, userRepo UserRepository) *MenuService {
	return &MenuService{
		menuRepo: menuRepo,
		userRepo: userRepo,
	}
}

// SetPermissionCacheInvalidator wires RBAC permission cache invalidation (optional).
func (s *MenuService) SetPermissionCacheInvalidator(p PermissionCacheInvalidator) {
	s.permInv = p
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
	log.Printf("[DEBUG] ========== GetUserMenus called for userID: %d ==========", userID)

	// Get user with roles
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		log.Printf("[ERROR] Failed to find user by ID %d: %v", userID, err)
		return nil, err
	}
	log.Printf("[DEBUG] Found user: %d, roles count: %d, IsSuperAdmin: %v", user.ID, len(user.Roles), user.IsSuperAdmin)

	// Check if user is super admin - use the IsSuperAdmin field from user model
	if user.IsSuperAdmin {
		log.Printf("[DEBUG] ========== User %d is super admin, getting ALL visible menus ==========", userID)

		// Get all menus first, then filter visible and enabled ones
		allMenus, err := s.menuRepo.FindAll(ctx)
		if err != nil {
			log.Printf("[ERROR] Failed to find all menus: %v", err)
			return nil, err
		}
		log.Printf("[DEBUG] Found %d total menus", len(allMenus))

		// Filter for visible and enabled menus
		var menus []models.Menu
		for _, menu := range allMenus {
			log.Printf("[DEBUG] Checking menu ID=%d, Name=%s, Title=%s, ParentID=%v, Visible=%d, Status=%d",
				menu.ID, menu.Name, menu.Title, menu.ParentID, menu.Visible, menu.Status)
			if menu.Visible == 1 && menu.Status == 1 {
				menus = append(menus, menu)
				log.Printf("[DEBUG] ✅ Added menu ID=%d (%s) to visible list", menu.ID, menu.Title)
			} else {
				log.Printf("[DEBUG] ❌ Skipped menu ID=%d (%s) - Visible=%d, Status=%d",
					menu.ID, menu.Title, menu.Visible, menu.Status)
			}
		}
		log.Printf("[DEBUG] Filtered to %d visible and enabled menus for super admin", len(menus))

		// Log all filtered menus
		for i, menu := range menus {
			log.Printf("[DEBUG] Visible menu %d: ID=%d, Name=%s, Title=%s, ParentID=%v",
				i+1, menu.ID, menu.Name, menu.Title, menu.ParentID)
		}

		// If no visible menus found, create a default dashboard menu
		if len(menus) == 0 {
			log.Printf("[WARN] No visible menus found, creating default menu")
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

		result := s.buildMenuRoutes(menus)
		log.Printf("[DEBUG] Built %d menu routes for super admin", len(result))
		for i, route := range result {
			log.Printf("[DEBUG] Root route %d: ID=%d, Name=%s, Title=%s, Children=%d",
				i+1, route.ID, route.Name, route.Meta.Title, len(route.Children))
			for j, child := range route.Children {
				log.Printf("[DEBUG]   Child route %d.%d: ID=%d, Name=%s, Title=%s",
					i+1, j+1, child.ID, child.Name, child.Meta.Title)
			}
		}
		log.Printf("[DEBUG] ========== Returning %d routes for super admin ==========", len(result))
		return result, nil
	}

	log.Printf("[DEBUG] ========== User %d is NOT super admin, using role-based access ==========", userID)
	// Extract role IDs for non-super-admin users
	roleIDs := make([]uint, 0)
	for _, role := range user.Roles {
		if role.Status == 1 { // Only include active roles
			roleIDs = append(roleIDs, role.ID)
		}
	}
	log.Printf("[DEBUG] Active role IDs for user %d: %v", userID, roleIDs)

	// If user has no active roles, return empty menu list
	if len(roleIDs) == 0 {
		log.Printf("[WARN] User %d has no active roles", userID)
		return []MenuRouteItem{}, nil
	}

	// Get menus by role IDs
	menus, err := s.menuRepo.FindByRoleIDs(ctx, roleIDs)
	if err != nil {
		log.Printf("[ERROR] Failed to find menus by role IDs %v: %v", roleIDs, err)
		return nil, err
	}
	log.Printf("[DEBUG] Found %d menus for roles %v", len(menus), roleIDs)

	// Filter visible and enabled menus
	var visibleMenus []models.Menu
	for _, menu := range menus {
		if menu.Visible == 1 && menu.Status == 1 {
			visibleMenus = append(visibleMenus, menu)
		}
	}
	log.Printf("[DEBUG] Filtered to %d visible and enabled menus", len(visibleMenus))

	// Build menu tree
	result := s.buildMenuRoutes(visibleMenus)
	log.Printf("[DEBUG] Built menu tree with %d root items", len(result))

	return result, nil
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
	log.Printf("[DEBUG] ========== buildMenuRoutes called with %d menus ==========", len(menus))

	menuMap := make(map[uint]*MenuRouteItem)
	var rootMenus []*MenuRouteItem

	// First pass: create menu items
	log.Printf("[DEBUG] First pass: creating menu items...")
	for _, menu := range menus {
		log.Printf("[DEBUG] Processing menu ID=%d, Name=%s, Title=%s, ParentID=%v",
			menu.ID, menu.Name, menu.Title, menu.ParentID)

		// Parse meta JSON string to extract metadata
		var metaData MenuMeta
		if menu.Meta != "" {
			// Try to parse the JSON meta string
			if err := json.Unmarshal([]byte(menu.Meta), &metaData); err != nil {
				log.Printf("[WARN] Failed to parse meta JSON for menu %s: %v", menu.Name, err)
				// Use default meta values
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
		log.Printf("[DEBUG] ✅ Created menu item ID=%d (%s) in map", menu.ID, menu.Title)
	}

	// Second pass: build tree structure
	log.Printf("[DEBUG] Second pass: building tree structure...")
	log.Printf("[DEBUG] MenuMap contains %d items", len(menuMap))

	for _, menu := range menus {
		item := menuMap[menu.ID]
		log.Printf("[DEBUG] Processing menu ID=%d (%s) for tree building...", menu.ID, menu.Title)

		if menu.ParentID == nil {
			// Root menu
			rootMenus = append(rootMenus, item)
			log.Printf("[DEBUG] ✅ Added ID=%d (%s) as ROOT menu", menu.ID, menu.Title)
		} else {
			// Child menu
			parentID := *menu.ParentID
			log.Printf("[DEBUG] Looking for parent ID=%d for child ID=%d (%s)", parentID, menu.ID, menu.Title)

			if parent, exists := menuMap[parentID]; exists {
				parent.Children = append(parent.Children, *item)
				log.Printf("[DEBUG] ✅ Added ID=%d (%s) as CHILD of ID=%d", menu.ID, menu.Title, parentID)
			} else {
				log.Printf("[DEBUG] ⚠️  Parent ID=%d not found for child ID=%d (%s), skipping",
					parentID, menu.ID, menu.Title)
			}
		}
	}

	// Sort children by sort field
	for _, root := range rootMenus {
		sort.Slice(root.Children, func(i, j int) bool {
			return root.Children[i].Sort < root.Children[j].Sort
		})
		log.Printf("[DEBUG] Sorted %d children for root menu ID=%d (%s)", len(root.Children), root.ID, root.Meta.Title)
	}

	// Sort root menus by sort field
	sort.Slice(rootMenus, func(i, j int) bool {
		return rootMenus[i].Sort < rootMenus[j].Sort
	})
	log.Printf("[DEBUG] Sorted %d root menus", len(rootMenus))

	// Convert pointer slice to value slice for return
	result := make([]MenuRouteItem, len(rootMenus))
	for i, root := range rootMenus {
		result[i] = *root
	}

	log.Printf("[DEBUG] ========== Final tree structure: %d root menus ==========", len(result))
	for i, root := range result {
		log.Printf("[DEBUG] Root menu %d: ID=%d, Title=%s, Sort=%d, Children=%d",
			i+1, root.ID, root.Meta.Title, root.Sort, len(root.Children))
		for j, child := range root.Children {
			log.Printf("[DEBUG]   Child %d.%d: ID=%d, Title=%s, Sort=%d",
				i+1, j+1, child.ID, child.Meta.Title, child.Sort)
		}
	}
	log.Printf("[DEBUG] ========== buildMenuRoutes completed ==========")

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
		log.Printf("[WARN] Failed to marshal meta to JSON: %v", err)
		return ""
	}
	return string(jsonData)
}
