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
