package repositories_test

import (
	"context"
	"testing"
	"time"

	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/repositories"
	"go-admin-scaffold/internal/core/types"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUserDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:user_repo_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Role{}, &models.UserRole{}))
	return db
}

func TestUserRepository_CRUD(t *testing.T) {
	db := setupUserDB(t)
	repo := repositories.NewUserRepository(db)
	cfg := &config.Config{}
	cfg.SuperAdminIDs = []uint{1}
	repo.SetConfig(cfg)
	ctx := context.Background()

	role := &models.Role{Name: "User", Code: "user", Status: 1}
	require.NoError(t, db.Create(role).Error)

	u := &models.User{Username: "alice", Password: "hash", Email: "a@b.c", Status: 1}
	require.NoError(t, repo.CreateWithRoles(ctx, u, []uint{role.ID}))
	require.NotZero(t, u.ID)

	byName, err := repo.FindByUsername(ctx, "alice")
	require.NoError(t, err)
	require.Equal(t, u.ID, byName.ID)

	byEmail, err := repo.FindByEmail(ctx, "a@b.c")
	require.NoError(t, err)
	require.Equal(t, u.ID, byEmail.ID)

	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, "alice", got.Username)
	if u.ID == 1 {
		require.True(t, got.IsSuperAdmin)
	}

	basic, err := repo.FindBasicByID(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, "alice", basic.Username)

	require.NoError(t, repo.UpdateLastLogin(ctx, u.ID))
	require.NoError(t, repo.UpdateStatus(ctx, u.ID, 0))
	require.NoError(t, repo.UpdateFields(ctx, u.ID, map[string]interface{}{"nickname": "A"}))

	page := &models.Pagination{Page: 1, PageSize: 10}
	list, err := repo.ListWithRoles(ctx, page)
	require.NoError(t, err)
	require.NotEmpty(t, list)

	status := 0
	filtered, err := repo.ListWithFilters(ctx, page, &types.UserSearchFilters{Username: "ali", Status: &status})
	require.NoError(t, err)
	require.NotEmpty(t, filtered)

	exported, err := repo.ExportWithFilters(ctx, &types.UserExportFilters{Username: "ali"})
	require.NoError(t, err)
	require.NotEmpty(t, exported)

	require.NoError(t, repo.ReplaceUserRoles(ctx, u.ID, []uint{role.ID}))
	require.NoError(t, repo.Delete(ctx, u.ID))
}

func TestLogAndMenuRepositories(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:log_menu_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.LoginLog{}, &models.OperationLog{}, &models.Menu{}, &models.Role{}, &models.RoleMenu{}))

	ctx := context.Background()
	logs := repositories.NewLogRepository(db)
	require.NoError(t, logs.CreateLoginLog(ctx, &models.LoginLog{
		UserID: 1, Username: "a", Status: 1, LoginTime: models.CustomTime(time.Now()),
	}))
	require.NoError(t, logs.CreateOperationLog(ctx, &models.OperationLog{
		UserID: 1, Username: "a", Module: "user", Action: "create", OperationTime: models.CustomTime(time.Now()),
	}))
	page := &models.Pagination{Page: 1, PageSize: 10}
	ll, err := logs.ListLoginLogs(ctx, page, map[string]interface{}{"username": "a"})
	require.NoError(t, err)
	require.Len(t, ll, 1)
	ol, err := logs.ListOperationLogs(ctx, page, map[string]interface{}{"module": "user"})
	require.NoError(t, err)
	require.Len(t, ol, 1)
	hist, err := logs.GetLoginLogsByUserID(ctx, 1, 5)
	require.NoError(t, err)
	require.Len(t, hist, 1)

	menus := repositories.NewMenuRepository(db)
	m := &models.Menu{Name: "dash", Title: "Dash", Status: 1, Visible: 1, Sort: 1, Permission: "dashboard:view"}
	require.NoError(t, menus.Create(ctx, m))
	got, err := menus.FindByID(ctx, m.ID)
	require.NoError(t, err)
	require.Equal(t, "dash", got.Name)
	all, err := menus.FindAll(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, all)
	tree, err := menus.FindTree(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, tree)
	vis, err := menus.FindVisibleMenus(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, vis)
	roots, err := menus.FindByParentID(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, roots)
	maxSort, err := menus.GetMaxSort(ctx, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, maxSort, 0)

	role := &models.Role{Name: "R", Code: "r", Status: 1}
	require.NoError(t, db.Create(role).Error)
	require.NoError(t, menus.UpdateMenuRoles(ctx, m.ID, []uint{role.ID}))
	byRole, err := menus.FindByRoleIDs(ctx, []uint{role.ID})
	require.NoError(t, err)
	require.NotEmpty(t, byRole)

	m.Title = "Dashboard"
	require.NoError(t, menus.Update(ctx, m))
	require.NoError(t, menus.Delete(ctx, m.ID))
}

func TestRoleRepository_withMenus(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:role2_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Role{}, &models.Menu{}, &models.RoleMenu{}, &models.UserRole{}))

	repo := repositories.NewRoleRepository(db)
	ctx := context.Background()
	menu := models.Menu{Name: "m", Title: "M", Status: 1, Visible: 1}
	require.NoError(t, db.Create(&menu).Error)

	role := &models.Role{Name: "Ops", Code: "ops2", Status: 1}
	require.NoError(t, repo.CreateWithMenus(ctx, role, []uint{menu.ID}))
	got, err := repo.FindByIDWithMenus(ctx, role.ID)
	require.NoError(t, err)
	require.NotEmpty(t, got.Menus)

	ids := []uint{menu.ID}
	updated, err := repo.UpdateWithMenus(ctx, role.ID, &models.Role{Name: "OpsX", Code: "ops2", Status: 1}, &ids)
	require.NoError(t, err)
	require.Equal(t, "OpsX", updated.Name)

	require.NoError(t, repo.SetMenus(ctx, role.ID, []uint{menu.ID}))
	require.NoError(t, repo.DeleteWithAssociations(ctx, role.ID))
}
