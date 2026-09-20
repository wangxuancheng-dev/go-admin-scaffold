package repositories

import (
	"context"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/pkg/logger"

	"gorm.io/gorm"
)

type MenuRepository struct {
	*BaseRepository
}

func NewMenuRepository(db *gorm.DB) *MenuRepository {
	return &MenuRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// FindByID retrieves a menu by ID with its children
func (r *MenuRepository) FindByID(ctx context.Context, id uint) (*models.Menu, error) {
	var menu models.Menu
	err := r.db.WithContext(ctx).
		Preload("Children").
		Where("id = ?", id).
		First(&menu).Error
	if err != nil {
		return nil, err
	}
	return &menu, nil
}

// FindAll retrieves all menus with their relationships
func (r *MenuRepository) FindAll(ctx context.Context) ([]models.Menu, error) {
	var menus []models.Menu
	err := r.db.WithContext(ctx).
		Preload("Parent").
		Preload("Children").
		Order("CASE WHEN parent_id IS NULL THEN 0 ELSE 1 END, sort ASC, id ASC").
		Find(&menus).Error
	return menus, err
}

// FindByParentID retrieves menus by parent ID
func (r *MenuRepository) FindByParentID(ctx context.Context, parentID *uint) ([]models.Menu, error) {
	var menus []models.Menu
	query := r.db.WithContext(ctx).Order("sort ASC, id ASC")

	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}

	err := query.Find(&menus).Error
	return menus, err
}

// FindTree retrieves menu tree structure
func (r *MenuRepository) FindTree(ctx context.Context) ([]models.Menu, error) {
	var allMenus []models.Menu
	err := r.db.WithContext(ctx).
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort ASC, id ASC")
		}).
		Order("CASE WHEN parent_id IS NULL THEN 0 ELSE 1 END, sort ASC, id ASC").
		Find(&allMenus).Error
	if err != nil {
		logger.Error(ctx, "menu FindTree query failed", "error", err)
		return nil, err
	}

	var rootMenus []models.Menu
	for i := range allMenus {
		menu := &allMenus[i]
		if menu.ParentID == nil {
			rootMenus = append(rootMenus, *menu)
		}
	}

	return rootMenus, nil
}

// FindByRoleIDs retrieves menus accessible by given role IDs
func (r *MenuRepository) FindByRoleIDs(ctx context.Context, roleIDs []uint) ([]models.Menu, error) {
	var menuIDs []uint
	err := r.db.WithContext(ctx).
		Table("role_menus").
		Where("role_id IN ?", roleIDs).
		Distinct("menu_id").
		Pluck("menu_id", &menuIDs).Error
	if err != nil {
		logger.Error(ctx, "menu FindByRoleIDs pluck menu_id failed", "error", err, "role_ids", roleIDs)
		return nil, err
	}

	if len(menuIDs) == 0 {
		return []models.Menu{}, nil
	}

	var allMenus []models.Menu
	err = r.db.WithContext(ctx).
		Where("id IN ?", menuIDs).
		Order("sort ASC, id ASC").
		Find(&allMenus).Error
	if err != nil {
		logger.Error(ctx, "menu FindByRoleIDs find menus failed", "error", err, "menu_ids", menuIDs)
		return nil, err
	}

	var parentIDs []uint
	parentIDMap := make(map[uint]bool)
	for _, menu := range allMenus {
		if menu.ParentID != nil && !parentIDMap[*menu.ParentID] {
			parentIDs = append(parentIDs, *menu.ParentID)
			parentIDMap[*menu.ParentID] = true
		}
	}

	if len(parentIDs) > 0 {
		var parentMenus []models.Menu
		err = r.db.WithContext(ctx).
			Where("id IN ?", parentIDs).
			Order("sort ASC, id ASC").
			Find(&parentMenus).Error
		if err != nil {
			logger.Error(ctx, "menu FindByRoleIDs find parents failed", "error", err, "parent_ids", parentIDs)
			return nil, err
		}
		allMenus = append(allMenus, parentMenus...)
	}

	return allMenus, nil
}

// FindVisibleMenus retrieves all visible and enabled menus
func (r *MenuRepository) FindVisibleMenus(ctx context.Context) ([]models.Menu, error) {
	var menus []models.Menu
	err := r.db.WithContext(ctx).
		Where("visible = 1 AND status = 1").
		Order("sort ASC, id ASC").
		Find(&menus).Error
	if err != nil {
		logger.Error(ctx, "menu FindVisibleMenus find failed", "error", err)
		return nil, err
	}
	return menus, nil
}

// Create creates a new menu
func (r *MenuRepository) Create(ctx context.Context, menu *models.Menu) error {
	return r.db.WithContext(ctx).Create(menu).Error
}

// Update updates an existing menu
func (r *MenuRepository) Update(ctx context.Context, menu *models.Menu) error {
	return r.db.WithContext(ctx).Save(menu).Error
}

// Delete deletes a menu by ID and its role associations.
func (r *MenuRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("menu_id = ?", id).Delete(&models.RoleMenu{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Menu{}, id).Error
	})
}

// UpdateMenuRoles updates the roles associated with a menu.
func (r *MenuRepository) UpdateMenuRoles(ctx context.Context, menuID uint, roleIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("menu_id = ?", menuID).Delete(&models.RoleMenu{}).Error; err != nil {
			return err
		}
		if len(roleIDs) == 0 {
			return nil
		}
		roleMenus := make([]models.RoleMenu, 0, len(roleIDs))
		for _, roleID := range roleIDs {
			roleMenus = append(roleMenus, models.RoleMenu{
				MenuID: menuID,
				RoleID: roleID,
			})
		}
		return tx.Create(&roleMenus).Error
	})
}

// GetMaxSort returns the maximum sort value for a given parent.
func (r *MenuRepository) GetMaxSort(ctx context.Context, parentID *uint) (int, error) {
	var maxSort int
	query := r.db.WithContext(ctx).Model(&models.Menu{}).Select("COALESCE(MAX(sort), 0)")
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}
	err := query.Scan(&maxSort).Error
	return maxSort, err
}
