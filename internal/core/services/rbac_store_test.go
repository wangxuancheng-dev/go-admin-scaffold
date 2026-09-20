package services

import (
	"context"
	"encoding/json"
	"testing"

	"go-admin-scaffold/internal/core/models"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memRBACStore struct {
	adminUsers map[uint]bool
	perms      map[uint][]string
	active     []string
	roles      []models.Role
	visible    []models.Menu
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
	return m.roles, nil
}
func (m *memRBACStore) ListVisibleMenus(ctx context.Context) ([]models.Menu, error) {
	return m.visible, nil
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

func TestRBACService_HasAnyAndAllPermissions(t *testing.T) {
	store := &memRBACStore{
		perms:  map[uint][]string{3: {"a", "b"}},
		active: []string{"a", "b", "c"},
	}
	rbac := NewRBACService(store, nil, nil)

	ok, err := rbac.HasAnyPermission(context.Background(), 3, []string{"z", "b"})
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = rbac.HasAllPermissions(context.Background(), &models.User{ID: 3}, []string{"a", "b"})
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = rbac.HasAllPermissions(context.Background(), &models.User{ID: 3}, []string{"a", "z"})
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = rbac.HasAnyPermission(context.Background(), 3, nil)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestRBACService_GetUserPermissions_andRoles(t *testing.T) {
	store := &memRBACStore{
		perms:  map[uint][]string{4: {"p1"}},
		active: []string{"p1"},
		roles:  []models.Role{{ID: 1, Code: "editor", Status: 1}},
		visible: []models.Menu{{Name: "dash", Status: 1, Visible: 1}},
	}
	rbac := NewRBACService(store, nil, nil)

	perms, err := rbac.GetUserPermissions(context.Background(), 4)
	require.NoError(t, err)
	require.Contains(t, perms, "p1")

	roles, err := rbac.GetUserRoles(context.Background(), 4)
	require.NoError(t, err)
	require.Len(t, roles, 1)
}

func TestRBACService_redisCacheAndInvalidate(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	store := &memRBACStore{
		perms:  map[uint][]string{10: {"cached:perm"}},
		active: []string{"cached:perm"},
	}
	auth := NewAuthService(&stubUserRepo{}, nil, testAuthConfig("0123456789abcdef0123456789abcdef", nil))
	rbac := NewRBACService(store, auth, rdb)
	ctx := context.Background()

	perms, err := rbac.GetUserPermissions(ctx, 10)
	require.NoError(t, err)
	require.Contains(t, perms, "cached:perm")

	raw, err := rdb.Get(ctx, userPermCacheKey(10)).Bytes()
	require.NoError(t, err)
	var cached []string
	require.NoError(t, json.Unmarshal(raw, &cached))

	perms2, err := rbac.GetUserPermissions(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, perms, perms2)

	rbac.InvalidateUserPermissions(ctx, 10)
	rbac.InvalidateAllPermissions(ctx)
}
