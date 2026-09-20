package services

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/repositories"
	"go-admin-scaffold/internal/core/types"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setupUserServiceDB(t *testing.T) (*gorm.DB, *repositories.UserRepository) {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", name)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{}, &models.Role{}, &models.UserRole{}, &models.Menu{}, &models.RoleMenu{},
		&models.LoginLog{}, &models.OperationLog{},
	))
	userRole := models.Role{Name: "User", Code: "user", Status: 1}
	editorRole := models.Role{Name: "Editor", Code: "editor", Status: 1}
	require.NoError(t, db.Create(&userRole).Error)
	require.NoError(t, db.Create(&editorRole).Error)
	repo := repositories.NewUserRepository(db)
	return db, repo
}

func TestUserService_CreateListDelete(t *testing.T) {
	_, userRepo := setupUserServiceDB(t)
	ctx := context.Background()

	logRepo := &stubLogRepo{}
	auth := NewAuthService(userRepo, NewLogService(logRepo), testAuthConfig("0123456789abcdef0123456789abcdef", nil))
	svc := NewUserService(userRepo, NewLogService(logRepo), testAuthConfig("0123456789abcdef0123456789abcdef", nil), auth, nil)

	created, err := svc.Create(ctx, &CreateUserRequest{
		Username: "newbie",
		Password: "password123",
		Email:    "newbie@example.com",
		Nickname: "New",
		Status:   1,
	})
	require.NoError(t, err)
	require.NotZero(t, created.ID)

	_, err = svc.Create(ctx, &CreateUserRequest{
		Username: "newbie", Password: "x", Email: "other@example.com", Status: 1,
	})
	assert.ErrorIs(t, err, ErrUsernameTaken)

	_, err = svc.Create(ctx, &CreateUserRequest{
		Username: "other", Password: "x", Email: "newbie@example.com", Status: 1,
	})
	assert.ErrorIs(t, err, ErrEmailTaken)

	page := &models.Pagination{Page: 1, PageSize: 10}
	list, err := svc.List(ctx, page)
	require.NoError(t, err)
	require.NotEmpty(t, list)

	hash, _ := bcrypt.GenerateFromPassword([]byte("x"), bcrypt.DefaultCost)
	toDelete := &models.User{Username: "gone", Email: "gone@ex.com", Password: string(hash), Status: 1}
	require.NoError(t, userRepo.CreateWithRoles(ctx, toDelete, nil))

	require.NoError(t, svc.Delete(ctx, toDelete.ID))
	err = svc.Delete(ctx, toDelete.ID)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserService_IsSuperAdminViaConfig(t *testing.T) {
	_, userRepo := setupUserServiceDB(t)
	cfg := &config.Config{SuperAdminIDs: []uint{99}}
	svc := NewUserService(userRepo, nil, cfg, nil, nil)
	assert.True(t, svc.IsSuperAdmin(99))
	assert.False(t, svc.IsSuperAdmin(1))
}

func TestUserService_extendedCRUD(t *testing.T) {
	db, userRepo := setupUserServiceDB(t)
	ctx := context.Background()
	logRepo := &stubLogRepo{}
	auth := NewAuthService(userRepo, NewLogService(logRepo), testAuthConfig("0123456789abcdef0123456789abcdef", nil))
	svc := NewUserService(userRepo, NewLogService(logRepo), testAuthConfig("0123456789abcdef0123456789abcdef", nil), auth, nil)

	created, err := svc.Create(ctx, &CreateUserRequest{
		Username: "worker", Password: "password123", Email: "w@ex.com", Status: 1,
	})
	require.NoError(t, err)

	got, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "worker", got.Username)

	status := 1
	list, err := svc.ListWithFilters(ctx, &models.Pagination{Page: 1, PageSize: 10}, &types.UserSearchFilters{
		Username: "worker", Status: &status,
	})
	require.NoError(t, err)
	require.Len(t, list, 1)

	_, err = svc.Update(ctx, created.ID, &UpdateUserRequest{Nickname: "W"})
	require.NoError(t, err)

	require.NoError(t, svc.ChangePassword(ctx, created.ID, &ChangePasswordRequest{
		OldPassword: "password123", NewPassword: "newpass99",
	}))

	require.NoError(t, svc.UpdateStatus(ctx, created.ID, 0))

	exported, err := svc.ExportUserList(ctx, &ExportUserListRequest{Username: "worker"})
	require.NoError(t, err)
	require.NotEmpty(t, exported)

	hist, err := svc.GetUserLoginHistory(ctx, created.ID, 5)
	require.NoError(t, err)
	require.Empty(t, hist)

	var editorRole models.Role
	require.NoError(t, db.Where("code = ?", "editor").First(&editorRole).Error)
	require.NoError(t, svc.UpdateUserRoles(ctx, created.ID, []uint{editorRole.ID}))
}

func TestUserService_superAdminProtected(t *testing.T) {
	db, userRepo := setupUserServiceDB(t)
	ctx := context.Background()
	cfg := testAuthConfig("0123456789abcdef0123456789abcdef", []uint{1})
	auth := NewAuthService(userRepo, nil, cfg)
	inv := &stubPermInv{}
	svc := NewUserService(userRepo, nil, cfg, auth, inv)

	hash, _ := bcrypt.GenerateFromPassword([]byte("x"), bcrypt.DefaultCost)
	admin := &models.User{Username: "root", Email: "root@ex.com", Password: string(hash), Status: 1}
	require.NoError(t, db.Create(admin).Error)
	cfg.SuperAdminIDs = []uint{admin.ID}

	err := svc.Delete(ctx, admin.ID)
	assert.ErrorIs(t, err, ErrSuperAdminDelete)

	_, err = svc.Update(ctx, admin.ID, &UpdateUserRequest{Username: "hacked"})
	assert.ErrorIs(t, err, ErrSuperAdminModify)

	_, err = svc.Update(ctx, admin.ID, &UpdateUserRequest{Nickname: "Root"})
	require.NoError(t, err)

	before := len(inv.users)
	svc.invalidateUserPermissions(ctx, admin.ID)
	require.Equal(t, before+1, len(inv.users))
}
