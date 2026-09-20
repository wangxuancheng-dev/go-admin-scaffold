package services

import (
	"context"
	"testing"

	"go-admin-scaffold/internal/core/models"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type stubMenuRepo struct {
	menus map[uint]*models.Menu
	next  uint
}

func newStubMenuRepo() *stubMenuRepo {
	return &stubMenuRepo{menus: make(map[uint]*models.Menu), next: 1}
}

func (s *stubMenuRepo) FindByID(ctx context.Context, id uint) (*models.Menu, error) {
	m, ok := s.menus[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *m
	return &cp, nil
}

func (s *stubMenuRepo) FindAll(ctx context.Context) ([]models.Menu, error) {
	out := make([]models.Menu, 0, len(s.menus))
	for _, m := range s.menus {
		out = append(out, *m)
	}
	return out, nil
}

func (s *stubMenuRepo) FindByParentID(ctx context.Context, parentID *uint) ([]models.Menu, error) {
	return nil, nil
}

func (s *stubMenuRepo) FindTree(ctx context.Context) ([]models.Menu, error) {
	return s.FindAll(ctx)
}

func (s *stubMenuRepo) FindByRoleIDs(ctx context.Context, roleIDs []uint) ([]models.Menu, error) {
	return nil, nil
}

func (s *stubMenuRepo) FindVisibleMenus(ctx context.Context) ([]models.Menu, error) {
	return s.FindAll(ctx)
}

func (s *stubMenuRepo) Create(ctx context.Context, menu *models.Menu) error {
	menu.ID = s.next
	s.next++
	cp := *menu
	s.menus[menu.ID] = &cp
	return nil
}

func (s *stubMenuRepo) Update(ctx context.Context, menu *models.Menu) error {
	s.menus[menu.ID] = menu
	return nil
}

func (s *stubMenuRepo) Delete(ctx context.Context, id uint) error {
	delete(s.menus, id)
	return nil
}

func (s *stubMenuRepo) UpdateMenuRoles(ctx context.Context, menuID uint, roleIDs []uint) error {
	return nil
}

func (s *stubMenuRepo) GetMaxSort(ctx context.Context, parentID *uint) (int, error) {
	return 0, nil
}

func TestMenuService_createGetDelete(t *testing.T) {
	menuRepo := newStubMenuRepo()
	userRepo := &stubUserRepo{user: &models.User{ID: 1, Username: "admin", IsSuperAdmin: true}}
	inv := &stubPermInv{}
	svc := NewMenuService(menuRepo, userRepo, inv)
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateMenuRequest{
		Name: "dash", Title: "Dashboard", Path: "/dashboard", Status: 1, Visible: 1, Type: 1,
	})
	require.NoError(t, err)
	require.NotZero(t, created.ID)
	require.Equal(t, 1, inv.all)

	got, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "dash", got.Name)

	all, err := svc.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)

	tree, err := svc.GetTree(ctx)
	require.NoError(t, err)
	require.Len(t, tree, 1)

	routes, err := svc.GetUserMenus(ctx, 1)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	require.NoError(t, svc.Delete(ctx, created.ID))
	require.Equal(t, 2, inv.all)
}
