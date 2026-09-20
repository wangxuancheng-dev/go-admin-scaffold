package services

import (
	"context"
	"testing"

	"go-admin-scaffold/internal/core/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memRBACStore struct {
	adminUsers map[uint]bool
	perms      map[uint][]string
	active     []string
}

func (m *memRBACStore) ListActivePermissions(ctx context.Context) ([]string, error) {
	return m.active, nil
}
func (m *memRBACStore) UserHasAdminRole(ctx context.Context, userID uint) (bool, error) {
	return m.adminUsers[userID], nil
}
func (m *memRBACStore) ListUserPermissions(ctx context.Context, userID uint) ([]string, error) {
	return m.perms[userID], nil
}
func (m *memRBACStore) ListUserRolesWithMenus(ctx context.Context, userID uint) ([]models.Role, error) {
	return nil, nil
}
func (m *memRBACStore) ListVisibleMenus(ctx context.Context) ([]models.Menu, error) {
	return nil, nil
}

func TestRBACService_CheckPermission_fromStore(t *testing.T) {
	store := &memRBACStore{
		adminUsers: map[uint]bool{},
		perms:      map[uint][]string{5: {"user:view", "todo:edit"}},
		active:     []string{"user:view", "user:create", "todo:edit"},
	}
	rbac := NewRBACService(store, nil, nil)

	ok, err := rbac.CheckPermission(context.Background(), &models.User{ID: 5}, "user:view")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = rbac.CheckPermission(context.Background(), &models.User{ID: 5}, "user:delete")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestRBACService_adminRoleBypass(t *testing.T) {
	store := &memRBACStore{adminUsers: map[uint]bool{8: true}}
	rbac := NewRBACService(store, nil, nil)
	ok, err := rbac.CheckPermission(context.Background(), &models.User{ID: 8}, "any")
	require.NoError(t, err)
	assert.True(t, ok)
}
