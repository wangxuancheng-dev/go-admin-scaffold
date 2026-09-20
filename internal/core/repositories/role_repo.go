package repositories

import (
	"context"
	"errors"

	"go-admin-scaffold/internal/core/models"

	"gorm.io/gorm"
)

// RoleRepository handles role persistence and role-menu associations.
type RoleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// DB exposes the underlying connection for transactional helpers.
func (r *RoleRepository) DB() *gorm.DB {
	return r.db
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

func (r *RoleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Role{}, id).Error
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
