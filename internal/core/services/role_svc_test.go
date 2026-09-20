package services

import (
	"context"
	"testing"

	"go-admin-scaffold/internal/core/models"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type stubRoleRepo struct {
	roles map[uint]*models.Role
	menus map[uint][]models.Menu
	next  uint
}

func newStubRoleRepo() *stubRoleRepo {
	return &stubRoleRepo{
		roles: make(map[uint]*models.Role),
		menus: make(map[uint][]models.Menu),
		next:  1,
	}
}

func (s *stubRoleRepo) ListWithMenus(ctx context.Context, pagination *models.Pagination) ([]models.Role, error) {
	out := make([]models.Role, 0, len(s.roles))
	for _, r := range s.roles {
		out = append(out, *r)
	}
	pagination.Total = int64(len(out))
	return out, nil
}

func (s *stubRoleRepo) FindByIDWithMenus(ctx context.Context, id uint) (*models.Role, error) {
	r, ok := s.roles[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *r
	return &cp, nil
}

func (s *stubRoleRepo) CreateWithMenus(ctx context.Context, role *models.Role, menuIDs []uint) error {
	role.ID = s.next
	s.next++
	cp := *role
	s.roles[role.ID] = &cp
	_ = menuIDs
	return nil
}

func (s *stubRoleRepo) UpdateWithMenus(ctx context.Context, id uint, role *models.Role, menuIDs *[]uint) (*models.Role, error) {
	existing, ok := s.roles[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	existing.Name = role.Name
	existing.Code = role.Code
	existing.Description = role.Description
	existing.Status = role.Status
	_ = menuIDs
	cp := *existing
	return &cp, nil
}

func (s *stubRoleRepo) DeleteWithAssociations(ctx context.Context, id uint) error {
	if _, ok := s.roles[id]; !ok {
		return gorm.ErrRecordNotFound
	}
	delete(s.roles, id)
	delete(s.menus, id)
	return nil
}

func (s *stubRoleRepo) ListMenusByRoleID(ctx context.Context, roleID uint) ([]models.Menu, error) {
	return s.menus[roleID], nil
}

func (s *stubRoleRepo) SetMenus(ctx context.Context, roleID uint, menuIDs []uint) error {
	menus := make([]models.Menu, 0, len(menuIDs))
	for _, id := range menuIDs {
		menus = append(menus, models.Menu{ID: id, Status: 1})
	}
	s.menus[roleID] = menus
	return nil
}

type stubPermInv struct {
	all    int
	users  []uint
}

func (s *stubPermInv) InvalidateUserPermissions(ctx context.Context, userID uint) {
	s.users = append(s.users, userID)
}

func (s *stubPermInv) InvalidateAllPermissions(ctx context.Context) {
	s.all++
}

func TestRoleService_CRUDAndInvalidate(t *testing.T) {
	repo := newStubRoleRepo()
	inv := &stubPermInv{}
	svc := NewRoleService(repo, inv)
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateRoleRequest{Name: "Ops", Code: "ops", Status: 1, MenuIDs: []uint{1}})
	require.NoError(t, err)
	require.Equal(t, 1, inv.all)

	list, err := svc.List(ctx, &models.Pagination{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, list, 1)

	got, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "ops", got.Code)

	menuIDs := []uint{1, 2}
	updated, err := svc.Update(ctx, created.ID, &UpdateRoleRequest{Name: "Ops2", Code: "ops", MenuIDs: menuIDs})
	require.NoError(t, err)
	require.Equal(t, "Ops2", updated.Name)
	require.Equal(t, 2, inv.all)

	require.NoError(t, svc.UpdateMenus(ctx, created.ID, &UpdateRoleMenusRequest{MenuIDs: []uint{3}}))
	menus, err := svc.GetMenus(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, menus, 1)
	require.Equal(t, 3, inv.all)

	require.NoError(t, svc.Delete(ctx, created.ID))
	require.Equal(t, 4, inv.all)
}
