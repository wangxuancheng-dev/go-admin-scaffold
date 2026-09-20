package services

import (
	"context"
	"errors"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/repositories"

	"gorm.io/gorm"
)

type RoleService struct {
	repo    *repositories.RoleRepository
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

func NewRoleService(repo *repositories.RoleRepository, permInv PermissionCacheInvalidator) *RoleService {
	return &RoleService{repo: repo, permInv: permInv}
}

func (s *RoleService) invalidateAllPermissions(ctx context.Context) {
	if s.permInv != nil {
		s.permInv.InvalidateAllPermissions(ctx)
	}
}

func (s *RoleService) List(ctx context.Context, pagination *models.Pagination) ([]models.Role, error) {
	return s.repo.ListWithMenus(ctx, pagination)
}

func (s *RoleService) Create(ctx context.Context, req *CreateRoleRequest) (*models.Role, error) {
	var result *models.Role
	err := s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		role := &models.Role{
			Name:        req.Name,
			Code:        req.Code,
			Description: req.Description,
			Status:      req.Status,
		}
		if err := tx.Create(role).Error; err != nil {
			return err
		}
		if len(req.MenuIDs) > 0 {
			if err := s.repo.ReplaceMenus(tx, role.ID, req.MenuIDs); err != nil {
				return err
			}
		}
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
	return s.repo.FindByIDWithMenus(ctx, id)
}

func (s *RoleService) Update(ctx context.Context, id uint, req *UpdateRoleRequest) (*models.Role, error) {
	var result *models.Role
	err := s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role models.Role
		if err := tx.First(&role, id).Error; err != nil {
			return err
		}
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
		if req.MenuIDs != nil {
			if err := s.repo.ReplaceMenus(tx, role.ID, req.MenuIDs); err != nil {
				return err
			}
		}
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
	err := s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role models.Role
		if err := tx.First(&role, id).Error; err != nil {
			return err
		}
		if role.Code == "admin" {
			return repositories.ErrAdminRoleProtected
		}
		if err := s.repo.DeleteRoleMenus(tx, id); err != nil {
			return err
		}
		if err := s.repo.DeleteUserRolesByRoleID(tx, id); err != nil {
			return err
		}
		return tx.Delete(&models.Role{}, id).Error
	})
	if err == nil {
		s.invalidateAllPermissions(ctx)
	}
	if errors.Is(err, repositories.ErrAdminRoleProtected) {
		return err
	}
	return err
}

func (s *RoleService) GetMenus(ctx context.Context, roleID uint) ([]models.Menu, error) {
	return s.repo.ListMenusByRoleID(ctx, roleID)
}

func (s *RoleService) UpdateMenus(ctx context.Context, roleID uint, req *UpdateRoleMenusRequest) error {
	err := s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role models.Role
		if err := tx.First(&role, roleID).Error; err != nil {
			return err
		}
		return s.repo.ReplaceMenus(tx, roleID, req.MenuIDs)
	})
	if err == nil {
		s.invalidateAllPermissions(ctx)
	}
	return err
}
