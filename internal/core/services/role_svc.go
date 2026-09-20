package services

import (
	"context"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/repositories"
)

// ErrAdminRoleProtected is returned when deleting the protected admin role.
var ErrAdminRoleProtected = repositories.ErrAdminRoleProtected

type RoleService struct {
	repo    RoleRepository
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

func NewRoleService(repo RoleRepository, permInv PermissionCacheInvalidator) *RoleService {
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
	role := &models.Role{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Status:      req.Status,
	}
	if err := s.repo.CreateWithMenus(ctx, role, req.MenuIDs); err != nil {
		return nil, err
	}
	s.invalidateAllPermissions(ctx)
	return role, nil
}

func (s *RoleService) GetByID(ctx context.Context, id uint) (*models.Role, error) {
	return s.repo.FindByIDWithMenus(ctx, id)
}

func (s *RoleService) Update(ctx context.Context, id uint, req *UpdateRoleRequest) (*models.Role, error) {
	role := &models.Role{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Status:      req.Status,
	}
	var menuIDs *[]uint
	if req.MenuIDs != nil {
		menuIDs = &req.MenuIDs
	}
	result, err := s.repo.UpdateWithMenus(ctx, id, role, menuIDs)
	if err != nil {
		return nil, err
	}
	if req.MenuIDs != nil {
		s.invalidateAllPermissions(ctx)
	}
	return result, nil
}

func (s *RoleService) Delete(ctx context.Context, id uint) error {
	err := s.repo.DeleteWithAssociations(ctx, id)
	if err == nil {
		s.invalidateAllPermissions(ctx)
	}
	return err
}

func (s *RoleService) GetMenus(ctx context.Context, roleID uint) ([]models.Menu, error) {
	return s.repo.ListMenusByRoleID(ctx, roleID)
}

func (s *RoleService) UpdateMenus(ctx context.Context, roleID uint, req *UpdateRoleMenusRequest) error {
	err := s.repo.SetMenus(ctx, roleID, req.MenuIDs)
	if err == nil {
		s.invalidateAllPermissions(ctx)
	}
	return err
}
