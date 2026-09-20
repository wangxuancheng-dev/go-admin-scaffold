package repositories

import (
	"context"

	"go-admin-scaffold/internal/core/models"

	"gorm.io/gorm"
)

// RBACRepository encapsulates permission / role-join queries used by RBACService.
type RBACRepository struct {
	db *gorm.DB
}

func NewRBACRepository(db *gorm.DB) *RBACRepository {
	return &RBACRepository{db: db}
}

func (r *RBACRepository) ListActivePermissions(ctx context.Context) ([]string, error) {
	var perms []string
	err := r.db.WithContext(ctx).Model(&models.Menu{}).
		Where("status = 1 AND visible = 1 AND permission != ''").
		Pluck("permission", &perms).Error
	return perms, err
}

func (r *RBACRepository) UserHasAdminRole(ctx context.Context, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("user_roles").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.code = 'admin' AND roles.status = 1", userID).
		Count(&count).Error
	return count > 0, err
}

func (r *RBACRepository) ListUserPermissions(ctx context.Context, userID uint) ([]string, error) {
	var permissions []string
	err := r.db.WithContext(ctx).Table("user_roles").
		Select("DISTINCT menus.permission").
		Joins("JOIN role_menus ON user_roles.role_id = role_menus.role_id").
		Joins("JOIN menus ON role_menus.menu_id = menus.id").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND menus.status = 1 AND menus.visible = 1 AND menus.permission != '' AND roles.status = 1", userID).
		Pluck("menus.permission", &permissions).Error
	return permissions, err
}

func (r *RBACRepository) ListUserRolesWithMenus(ctx context.Context, userID uint) ([]models.Role, error) {
	var roles []models.Role
	err := r.db.WithContext(ctx).
		Preload("Menus", "status = 1 AND visible = 1").
		Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND roles.status = 1", userID).
		Find(&roles).Error
	return roles, err
}

func (r *RBACRepository) ListVisibleMenus(ctx context.Context) ([]models.Menu, error) {
	var menus []models.Menu
	err := r.db.WithContext(ctx).Where("status = 1 AND visible = 1").Find(&menus).Error
	return menus, err
}
