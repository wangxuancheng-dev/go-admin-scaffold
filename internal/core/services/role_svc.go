package services

import (
	"app/internal/core/models"
	"context"
	"errors"

	"gorm.io/gorm"
)

type RoleService struct {
	db      *gorm.DB
	permInv PermissionCacheInvalidator
}

type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
	Status      int    `json:"status"`
	MenuIDs     []uint `json:"menu_ids"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Status      int    `json:"status"`
	MenuIDs     []uint `json:"menu_ids"`
}

type UpdateRoleMenusRequest struct {
	MenuIDs []uint `json:"menu_ids" binding:"required"`
}

func NewRoleService(db *gorm.DB) *RoleService {
	return &RoleService{
		db: db,
	}
}

// SetPermissionCacheInvalidator wires RBAC permission cache invalidation (optional).
func (s *RoleService) SetPermissionCacheInvalidator(p PermissionCacheInvalidator) {
	s.permInv = p
}

func (s *RoleService) invalidateAllPermissions(ctx context.Context) {
	if s.permInv != nil {
		s.permInv.InvalidateAllPermissions(ctx)
	}
}

func (s *RoleService) List(ctx context.Context, pagination *models.Pagination) ([]models.Role, error) {
	var roles []models.Role
	var total int64

	query := s.db.WithContext(ctx).Model(&models.Role{}).Preload("Menus")

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply pagination
	if pagination != nil {
		offset := (pagination.Page - 1) * pagination.PageSize
		query = query.Offset(offset).Limit(pagination.PageSize)
		pagination.Total = total
	}

	err := query.Find(&roles).Error
	return roles, err
}

func (s *RoleService) Create(ctx context.Context, req *CreateRoleRequest) (*models.Role, error) {
	var result *models.Role
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		role := &models.Role{
			Name:        req.Name,
			Code:        req.Code,
			Description: req.Description,
			Status:      req.Status,
		}

		// Create role
		if err := tx.Create(role).Error; err != nil {
			return err
		}

		// Assign menus
		if len(req.MenuIDs) > 0 {
			if err := s.assignMenus(tx, role.ID, req.MenuIDs); err != nil {
				return err
			}
		}

		// Load menus
		if err := tx.Preload("Menus").First(role, role.ID).Error; err != nil {
			return err
		}

		result = role
		return nil
	})

	if err == nil {
		s.invalidateAllPermissions(ctx)
	}
	return result, err
}

func (s *RoleService) GetByID(ctx context.Context, id uint) (*models.Role, error) {
	var role models.Role
	err := s.db.WithContext(ctx).Preload("Menus").First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *RoleService) Update(ctx context.Context, id uint, req *UpdateRoleRequest) (*models.Role, error) {
	var result *models.Role
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role models.Role
		if err := tx.First(&role, id).Error; err != nil {
			return err
		}

		// Update basic fields
		if req.Name != "" {
			role.Name = req.Name
		}
		if req.Code != "" {
			role.Code = req.Code
		}
		if req.Description != "" {
			role.Description = req.Description
		}
		if req.Status != 0 {
			role.Status = req.Status
		}

		if err := tx.Save(&role).Error; err != nil {
			return err
		}

		// Update menu associations if provided
		if req.MenuIDs != nil {
			if err := s.assignMenus(tx, role.ID, req.MenuIDs); err != nil {
				return err
			}
		}

		// Reload with menus
		if err := tx.Preload("Menus").First(&role, role.ID).Error; err != nil {
			return err
		}

		result = &role
		return nil
	})

	if err == nil && req.MenuIDs != nil {
		s.invalidateAllPermissions(ctx)
	}
	return result, err
}

func (s *RoleService) Delete(ctx context.Context, id uint) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查是否是超级管理员角色
		var role models.Role
		if err := tx.First(&role, id).Error; err != nil {
			return err
		}

		if role.Code == "admin" {
			return errors.New("cannot delete admin role")
		}

		// Remove role-menu associations
		if err := tx.Where("role_id = ?", id).Delete(&models.RoleMenu{}).Error; err != nil {
			return err
		}

		// Remove user-role associations
		if err := tx.Exec("DELETE FROM user_roles WHERE role_id = ?", id).Error; err != nil {
			return err
		}

		// Delete role
		return tx.Delete(&models.Role{}, id).Error
	})
	if err == nil {
		s.invalidateAllPermissions(ctx)
	}
	return err
}

// assignMenus assigns menus to a role
func (s *RoleService) assignMenus(tx *gorm.DB, roleID uint, menuIDs []uint) error {
	// Remove existing associations
	if err := tx.Where("role_id = ?", roleID).Delete(&models.RoleMenu{}).Error; err != nil {
		return err
	}

	// Add new associations
	if len(menuIDs) > 0 {
		var roleMenus []models.RoleMenu
		for _, menuID := range menuIDs {
			roleMenus = append(roleMenus, models.RoleMenu{
				RoleID: roleID,
				MenuID: menuID,
			})
		}
		return tx.Create(&roleMenus).Error
	}

	return nil
}

// GetMenus returns all menus for a role
func (s *RoleService) GetMenus(ctx context.Context, roleID uint) ([]models.Menu, error) {
	var menus []models.Menu
	err := s.db.WithContext(ctx).
		Joins("JOIN role_menus ON menus.id = role_menus.menu_id").
		Where("role_menus.role_id = ? AND menus.status = 1", roleID).
		Find(&menus).Error
	return menus, err
}

// UpdateMenus updates the menus of a role
func (s *RoleService) UpdateMenus(ctx context.Context, roleID uint, req *UpdateRoleMenusRequest) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check if role exists
		var role models.Role
		if err := tx.First(&role, roleID).Error; err != nil {
			return err
		}

		// Update menu associations
		return s.assignMenus(tx, roleID, req.MenuIDs)
	})
	if err == nil {
		s.invalidateAllPermissions(ctx)
	}
	return err
}
