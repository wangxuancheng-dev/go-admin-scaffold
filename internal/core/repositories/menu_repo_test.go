package repositories_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/repositories"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupMenuRepoDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", name)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Role{}, &models.Menu{}, &models.RoleMenu{}))
	return db
}

func TestMenuRepository_CRUDAndQueries(t *testing.T) {
	db := setupMenuRepoDB(t)
	repo := repositories.NewMenuRepository(db)
	ctx := context.Background()

	root := &models.Menu{Name: "root", Title: "Root", Status: 1, Visible: 1, Permission: "root:view", Sort: 1}
	require.NoError(t, repo.Create(ctx, root))

	child := &models.Menu{Name: "child", Title: "Child", Status: 1, Visible: 1, ParentID: &root.ID, Sort: 2}
	require.NoError(t, repo.Create(ctx, child))

	got, err := repo.FindByID(ctx, root.ID)
	require.NoError(t, err)
	require.NotEmpty(t, got.Children)

	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(all), 2)

	roots, err := repo.FindByParentID(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, roots)

	children, err := repo.FindByParentID(ctx, &root.ID)
	require.NoError(t, err)
	require.Len(t, children, 1)

	tree, err := repo.FindTree(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, tree)

	hidden := &models.Menu{Name: "hidden", Title: "Hidden", Status: 1}
	require.NoError(t, repo.Create(ctx, hidden))
	require.NoError(t, db.Model(hidden).Update("visible", 0).Error)
	visible, err := repo.FindVisibleMenus(ctx)
	require.NoError(t, err)
	for _, m := range visible {
		require.NotEqual(t, "hidden", m.Name)
	}

	role := models.Role{Name: "R", Code: "r", Status: 1}
	require.NoError(t, db.Create(&role).Error)
	require.NoError(t, db.Create(&models.RoleMenu{RoleID: role.ID, MenuID: root.ID}).Error)

	byRole, err := repo.FindByRoleIDs(ctx, []uint{role.ID})
	require.NoError(t, err)
	require.NotEmpty(t, byRole)

	empty, err := repo.FindByRoleIDs(ctx, []uint{999})
	require.NoError(t, err)
	require.Empty(t, empty)

	maxSort, err := repo.GetMaxSort(ctx, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, maxSort, 1)

	maxChild, err := repo.GetMaxSort(ctx, &root.ID)
	require.NoError(t, err)
	require.Equal(t, 2, maxChild)

	child.Title = "Child Updated"
	require.NoError(t, repo.Update(ctx, child))

	require.NoError(t, repo.UpdateMenuRoles(ctx, root.ID, []uint{role.ID}))
	require.NoError(t, repo.UpdateMenuRoles(ctx, root.ID, nil))

	require.NoError(t, repo.Delete(ctx, child.ID))
}
