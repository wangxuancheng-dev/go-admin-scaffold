package repositories_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/repositories"
	"go-admin-scaffold/internal/core/types"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setupUserRepoDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", name)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{}, &models.Role{}, &models.Menu{}, &models.UserRole{}, &models.RoleMenu{},
	))
	return db
}

func seedUserRoles(t *testing.T, db *gorm.DB) (userRole, editorRole models.Role) {
	t.Helper()
	userRole = models.Role{Name: "User", Code: "user", Status: 1}
	editorRole = models.Role{Name: "Editor", Code: "editor", Status: 1}
	adminRole := models.Role{Name: "Admin", Code: "admin", Status: 1}
	require.NoError(t, db.Create(&userRole).Error)
	require.NoError(t, db.Create(&editorRole).Error)
	require.NoError(t, db.Create(&adminRole).Error)
	return userRole, editorRole
}

func TestUserRepository_findAndLogin(t *testing.T) {
	db := setupUserRepoDB(t)
	userRole, _ := seedUserRoles(t, db)
	repo := repositories.NewUserRepository(db)
	ctx := context.Background()

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)
	u := &models.User{Username: "alice", Email: "alice@example.com", Password: string(hash), Status: 1}
	require.NoError(t, repo.CreateWithRoles(ctx, u, []uint{userRole.ID}))
	repo.SetConfig(&config.Config{SuperAdminIDs: []uint{u.ID}})

	byName, err := repo.FindByUsername(ctx, "alice")
	require.NoError(t, err)
	require.Equal(t, u.ID, byName.ID)

	byEmail, err := repo.FindByEmail(ctx, "alice@example.com")
	require.NoError(t, err)
	require.Equal(t, "alice", byEmail.Username)

	require.NoError(t, repo.UpdateLastLogin(ctx, u.ID))
	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	require.NotNil(t, got.LastLoginAt)
}

func TestUserRepository_listExportAndFilters(t *testing.T) {
	db := setupUserRepoDB(t)
	userRole, editorRole := seedUserRoles(t, db)
	repo := repositories.NewUserRepository(db)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("x"), bcrypt.DefaultCost)
	u1 := &models.User{Username: "u1", Email: "u1@ex.com", Password: string(hash), Status: 1}
	u2 := &models.User{Username: "u2", Email: "u2@ex.com", Password: string(hash), Status: 0}
	require.NoError(t, repo.CreateWithRoles(ctx, u1, []uint{editorRole.ID}))
	require.NoError(t, repo.CreateWithRoles(ctx, u2, nil))

	page := &models.Pagination{Page: 1, PageSize: 10}
	list, err := repo.ListWithRoles(ctx, page)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list), 2)
	require.Equal(t, int64(2), page.Total)

	status := 1
	filtered, err := repo.ListWithFilters(ctx, page, &types.UserSearchFilters{
		Username: "u1",
		Status:   &status,
		RoleID:   editorRole.ID,
	})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	require.Equal(t, "u1", filtered[0].Username)

	emptyRole, err := repo.ListWithFilters(ctx, page, &types.UserSearchFilters{RoleID: 9999})
	require.NoError(t, err)
	require.Empty(t, emptyRole)
	require.Equal(t, int64(0), page.Total)

	exported, err := repo.ExportWithFilters(ctx, &types.UserExportFilters{Username: "u2"})
	require.NoError(t, err)
	require.Len(t, exported, 1)

	_ = userRole
}

func TestUserRepository_updateAndReplaceRoles(t *testing.T) {
	db := setupUserRepoDB(t)
	userRole, editorRole := seedUserRoles(t, db)
	repo := repositories.NewUserRepository(db)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("x"), bcrypt.DefaultCost)
	u := &models.User{Username: "bob", Email: "bob@ex.com", Password: string(hash), Status: 1}
	require.NoError(t, repo.CreateWithRoles(ctx, u, []uint{userRole.ID}))

	basic, err := repo.FindBasicByID(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, "bob", basic.Username)

	require.NoError(t, repo.UpdateFields(ctx, u.ID, map[string]interface{}{"nickname": "Bobby"}))
	require.NoError(t, repo.UpdateStatus(ctx, u.ID, 0))

	u.Nickname = "Bobby2"
	require.NoError(t, repo.Update(ctx, u))

	require.NoError(t, repo.ReplaceUserRoles(ctx, u.ID, []uint{editorRole.ID}))
	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, got.Roles, 1)
	require.Equal(t, "editor", got.Roles[0].Code)

	err = repo.ReplaceUserRoles(ctx, 99999, []uint{editorRole.ID})
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var adminRole models.Role
	require.NoError(t, db.Where("code = ?", "admin").First(&adminRole).Error)
	err = repo.ReplaceUserRoles(ctx, u.ID, []uint{adminRole.ID})
	require.ErrorIs(t, err, repositories.ErrCannotAssignAdminRole)

	require.NoError(t, repo.Delete(ctx, u.ID))
	_, err = repo.FindByID(ctx, u.ID)
	require.Error(t, err)
}

func TestUserRepository_createWithDefaultRole(t *testing.T) {
	db := setupUserRepoDB(t)
	seedUserRoles(t, db)
	repo := repositories.NewUserRepository(db)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("x"), bcrypt.DefaultCost)
	u := &models.User{Username: "carol", Email: "c@ex.com", Password: string(hash), Status: 1}
	require.NoError(t, repo.CreateWithRoles(ctx, u, nil))
	require.NotZero(t, u.ID)
	require.NotEmpty(t, u.Roles)
	require.Equal(t, "user", u.Roles[0].Code)

	require.NoError(t, repo.Create(ctx, &models.User{
		Username: "direct", Email: "d@ex.com", Password: string(hash), Status: 1,
	}))
}

func TestUserRepository_exportTimeFilters(t *testing.T) {
	db := setupUserRepoDB(t)
	seedUserRoles(t, db)
	repo := repositories.NewUserRepository(db)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("x"), bcrypt.DefaultCost)
	require.NoError(t, repo.Create(ctx, &models.User{
		Username: "timed", Email: "t@ex.com", Password: string(hash), Status: 1,
	}))

	now := time.Now()
	_, err := repo.ExportWithFilters(ctx, &types.UserExportFilters{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	})
	require.NoError(t, err)

	require.NoError(t, repo.UpdateFields(ctx, 99999, map[string]interface{}{}))
}
