package repositories_test

import (
	"context"
	"testing"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/repositories"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRoleRepository_CRUD(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:role_repo?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Role{}, &models.Menu{}, &models.RoleMenu{}))

	repo := repositories.NewRoleRepository(db)
	ctx := context.Background()

	role := &models.Role{Name: "Ops", Code: "ops", Status: 1}
	require.NoError(t, repo.Create(ctx, role))
	require.NotZero(t, role.ID)

	got, err := repo.FindByID(ctx, role.ID)
	require.NoError(t, err)
	require.Equal(t, "ops", got.Code)

	menu := models.Menu{Name: "m", Title: "M", Status: 1, Visible: 1, Permission: "x:y"}
	require.NoError(t, db.Create(&menu).Error)
	require.NoError(t, repo.ReplaceMenus(nil, role.ID, []uint{menu.ID}))

	menus, err := repo.ListMenusByRoleID(ctx, role.ID)
	require.NoError(t, err)
	require.Len(t, menus, 1)

	page := &models.Pagination{Page: 1, PageSize: 10}
	list, err := repo.ListWithMenus(ctx, page)
	require.NoError(t, err)
	require.NotEmpty(t, list)
}

func TestRoleRepository_withMenusAndAssociations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:role_menus?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Role{}, &models.Menu{}, &models.RoleMenu{}, &models.UserRole{}, &models.User{}))

	repo := repositories.NewRoleRepository(db)
	ctx := context.Background()

	menu1 := models.Menu{Name: "m1", Title: "M1", Status: 1, Visible: 1}
	menu2 := models.Menu{Name: "m2", Title: "M2", Status: 1, Visible: 1}
	require.NoError(t, db.Create(&menu1).Error)
	require.NoError(t, db.Create(&menu2).Error)

	role := &models.Role{Name: "Custom", Code: "custom", Status: 1}
	require.NoError(t, repo.CreateWithMenus(ctx, role, []uint{menu1.ID, menu2.ID}))
	require.Len(t, role.Menus, 2)

	withMenus, err := repo.FindByIDWithMenus(ctx, role.ID)
	require.NoError(t, err)
	require.Len(t, withMenus.Menus, 2)

	role.Name = "Custom Updated"
	updated, err := repo.UpdateWithMenus(ctx, role.ID, role, &[]uint{menu1.ID})
	require.NoError(t, err)
	require.Equal(t, "Custom Updated", updated.Name)
	require.Len(t, updated.Menus, 1)

	_, err = repo.UpdateWithMenus(ctx, role.ID, &models.Role{Name: "X"}, nil)
	require.NoError(t, err)

	require.NoError(t, repo.SetMenus(ctx, role.ID, []uint{menu2.ID}))
	got, err := repo.FindByIDWithMenus(ctx, role.ID)
	require.NoError(t, err)
	require.Len(t, got.Menus, 1)

	user := models.User{Username: "u", Email: "u@ex.com", Password: "x", Status: 1}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&models.UserRole{UserID: user.ID, RoleID: role.ID}).Error)

	require.NoError(t, repo.DeleteWithAssociations(ctx, role.ID))
	_, err = repo.FindByID(ctx, role.ID)
	require.Error(t, err)

	admin := models.Role{Name: "Admin", Code: "admin", Status: 1}
	require.NoError(t, db.Create(&admin).Error)
	err = repo.DeleteWithAssociations(ctx, admin.ID)
	require.ErrorIs(t, err, repositories.ErrAdminRoleProtected)

	require.NoError(t, repo.Delete(ctx, admin.ID))
}
