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

func setupRBACDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Menu{},
		&models.UserRole{},
		&models.RoleMenu{},
	))
	return db
}

func TestRBACRepository_permissionQueries(t *testing.T) {
	db := setupRBACDB(t)
	repo := repositories.NewRBACRepository(db)
	ctx := context.Background()

	role := models.Role{Name: "Editor", Code: "editor", Status: 1}
	require.NoError(t, db.Create(&role).Error)
	menu := models.Menu{Name: "users", Title: "Users", Status: 1, Visible: 1, Permission: "user:view"}
	require.NoError(t, db.Create(&menu).Error)
	user := models.User{Username: "ed", Email: "ed@example.com", Password: "x", Status: 1}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&models.UserRole{UserID: user.ID, RoleID: role.ID}).Error)
	require.NoError(t, db.Create(&models.RoleMenu{RoleID: role.ID, MenuID: menu.ID}).Error)

	perms, err := repo.ListUserPermissions(ctx, user.ID)
	require.NoError(t, err)
	require.Contains(t, perms, "user:view")

	ok, err := repo.UserHasAdminRole(ctx, user.ID)
	require.NoError(t, err)
	require.False(t, ok)

	active, err := repo.ListActivePermissions(ctx)
	require.NoError(t, err)
	require.Contains(t, active, "user:view")
}
