package repositories

import (
	"context"
	"errors"

	"go-admin-scaffold/internal/core/models"

	"gorm.io/gorm"
)

// RoleRepository handles role persistence and role-menu associations.
type RoleRepository struct {
	*BaseRepository
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{BaseRepository: NewBaseRepository(db)}
}

func (r *RoleRepository) ListWithMenus(ctx context.Context, pagination *models.Pagination) ([]models.Role, error) {
	var roles []models.Role
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Role{}).Preload("Menus")
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if pagination != nil {
		offset := (pagination.Page - 1) * pagination.PageSize
		query = query.Offset(offset).Limit(pagination.PageSize)
		pagination.Total = total
	}

	err := query.Find(&roles).Error
	return roles, err
}

func (r *RoleRepository) Create(ctx context.Context, role *models.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

// CreateWithMenus creates a role and optionally assigns menus in a transaction.
// Menus are preloaded onto role on success.
func (r *RoleRepository) CreateWithMenus(ctx context.Context, role *models.Role, menuIDs []uint) error {
	return r.Transaction(ctx, func(tx *gorm.DB) error {
		if err := tx.Create(role).Error; err != nil {
			return err
		}
		if len(menuIDs) > 0 {
			if err := r.ReplaceMenus(tx, role.ID, menuIDs); err != nil {
				return err
			}
		}
		return tx.Preload("Menus").First(role, role.ID).Error
	})
}

func (r *RoleRepository) FindByID(ctx context.Context, id uint) (*models.Role, error) {
	var role models.Role
	if err := r.db.WithContext(ctx).First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) FindByIDWithMenus(ctx context.Context, id uint) (*models.Role, error) {
	var role models.Role
	if err := r.db.WithContext(ctx).Preload("Menus").First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) Update(ctx context.Context, role *models.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

// UpdateWithMenus updates a role and optionally replaces its menus in a transaction.
// If menuIDs is nil, menu associations are left unchanged; a non-nil (possibly empty) slice replaces them.
func (r *RoleRepository) UpdateWithMenus(ctx context.Context, id uint, role *models.Role, menuIDs *[]uint) (*models.Role, error) {
	var result *models.Role
	err := r.Transaction(ctx, func(tx *gorm.DB) error {
		var existing models.Role
		if err := tx.First(&existing, id).Error; err != nil {
			return err
		}
		if role.Name != "" {
			existing.Name = role.Name
		}
		if role.Code != "" {
			existing.Code = role.Code
		}
		if role.Description != "" {
			existing.Description = role.Description
		}
		if role.Status != 0 {
			existing.Status = role.Status
		}
		if err := tx.Save(&existing).Error; err != nil {
			return err
		}
		if menuIDs != nil {
			if err := r.ReplaceMenus(tx, existing.ID, *menuIDs); err != nil {
				return err
			}
		}
		if err := tx.Preload("Menus").First(&existing, existing.ID).Error; err != nil {
			return err
		}
		result = &existing
		return nil
	})
	return result, err
}

func (r *RoleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Role{}, id).Error
}

// DeleteWithAssociations deletes a role and its associations in a transaction.
// Returns ErrAdminRoleProtected when attempting to delete the admin role.
func (r *RoleRepository) DeleteWithAssociations(ctx context.Context, id uint) error {
	return r.Transaction(ctx, func(tx *gorm.DB) error {
		var role models.Role
		if err := tx.First(&role, id).Error; err != nil {
			return err
		}
		if role.Code == "admin" {
			return ErrAdminRoleProtected
		}
		if err := r.DeleteRoleMenus(tx, id); err != nil {
			return err
		}
		if err := r.DeleteUserRolesByRoleID(tx, id); err != nil {
			return err
		}
		return tx.Delete(&models.Role{}, id).Error
	})
}

// SetMenus verifies the role exists and replaces its menu associations in a transaction.
func (r *RoleRepository) SetMenus(ctx context.Context, roleID uint, menuIDs []uint) error {
	return r.Transaction(ctx, func(tx *gorm.DB) error {
		var role models.Role
		if err := tx.First(&role, roleID).Error; err != nil {
			return err
		}
		return r.ReplaceMenus(tx, roleID, menuIDs)
	})
}

// ReplaceMenus replaces all menu associations for a role inside an optional transaction.
func (r *RoleRepository) ReplaceMenus(tx *gorm.DB, roleID uint, menuIDs []uint) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	if err := db.Where("role_id = ?", roleID).Delete(&models.RoleMenu{}).Error; err != nil {
		return err
	}
	if len(menuIDs) == 0 {
		return nil
	}

	roleMenus := make([]models.RoleMenu, 0, len(menuIDs))
	for _, menuID := range menuIDs {
		roleMenus = append(roleMenus, models.RoleMenu{
			RoleID: roleID,
			MenuID: menuID,
		})
	}
	return db.Create(&roleMenus).Error
}

func (r *RoleRepository) DeleteRoleMenus(tx *gorm.DB, roleID uint) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.Where("role_id = ?", roleID).Delete(&models.RoleMenu{}).Error
}

func (r *RoleRepository) DeleteUserRolesByRoleID(tx *gorm.DB, roleID uint) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.Exec("DELETE FROM user_roles WHERE role_id = ?", roleID).Error
}

func (r *RoleRepository) ListMenusByRoleID(ctx context.Context, roleID uint) ([]models.Menu, error) {
	var menus []models.Menu
	err := r.db.WithContext(ctx).
		Joins("JOIN role_menus ON menus.id = role_menus.menu_id").
		Where("role_menus.role_id = ? AND menus.status = 1", roleID).
		Find(&menus).Error
	return menus, err
}

// ErrAdminRoleProtected is returned when attempting to delete the admin role.
var ErrAdminRoleProtected = errors.New("cannot delete admin role")
