package services

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/pkg/logger"
)

const (
	rbacPermKeyPrefix = "rbac:perm:"
	rbacPermTTL       = 5 * time.Minute
)

// RBACPermissionStore is the data access needed by RBACService.
type RBACPermissionStore interface {
	ListActivePermissions(ctx context.Context) ([]string, error)
	UserHasAdminRole(ctx context.Context, userID uint) (bool, error)
	ListUserPermissions(ctx context.Context, userID uint) ([]string, error)
	ListUserRolesWithMenus(ctx context.Context, userID uint) ([]models.Role, error)
	ListVisibleMenus(ctx context.Context) ([]models.Menu, error)
}

// RBACService handles role-based access control via RBACPermissionStore + optional Redis cache.
type RBACService struct {
	store   RBACPermissionStore
	authSvc AuthServiceInterface
	rdb     *redis.Client
}

// NewRBACService wires store, auth checker, and optional Redis cache in one shot.
func NewRBACService(store RBACPermissionStore, authSvc AuthServiceInterface, rdb *redis.Client) *RBACService {
	return &RBACService{
		store:   store,
		authSvc: authSvc,
		rdb:     rdb,
	}
}

func userPermCacheKey(userID uint) string {
	return rbacPermKeyPrefix + strconv.FormatUint(uint64(userID), 10)
}

// InvalidateUserPermissions implements PermissionCacheInvalidator.
func (s *RBACService) InvalidateUserPermissions(ctx context.Context, userID uint) {
	if s.rdb == nil {
		return
	}
	if err := s.rdb.Del(ctx, userPermCacheKey(userID)).Err(); err != nil {
		logger.Warn(ctx, "rbac cache: invalidate user", "user_id", userID, "error", err)
	}
}

// InvalidateAllPermissions implements PermissionCacheInvalidator.
func (s *RBACService) InvalidateAllPermissions(ctx context.Context) {
	if s.rdb == nil {
		return
	}
	iter := s.rdb.Scan(ctx, 0, rbacPermKeyPrefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		if err := s.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			logger.Warn(ctx, "rbac cache: invalidate all", "key", iter.Val(), "error", err)
		}
	}
	if err := iter.Err(); err != nil {
		logger.Warn(ctx, "rbac cache: scan keys", "error", err)
	}
}

func (s *RBACService) computeUserPermissions(ctx context.Context, userID uint) ([]string, error) {
	if s.authSvc != nil && s.authSvc.IsSuperAdmin(userID) {
		return s.store.ListActivePermissions(ctx)
	}

	isAdmin, err := s.store.UserHasAdminRole(ctx, userID)
	if err != nil {
		return nil, err
	}
	if isAdmin {
		return s.store.ListActivePermissions(ctx)
	}
	return s.store.ListUserPermissions(ctx, userID)
}

// CheckPermission checks if a user has the specified permission
func (s *RBACService) CheckPermission(ctx context.Context, user interface{}, permission string) (bool, error) {
	userModel, ok := user.(*models.User)
	if !ok {
		return false, nil
	}

	if s.authSvc != nil && s.authSvc.IsSuperAdmin(userModel.ID) {
		return true, nil
	}

	isAdmin, err := s.store.UserHasAdminRole(ctx, userModel.ID)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}

	perms, err := s.GetUserPermissions(ctx, userModel.ID)
	if err != nil {
		return false, err
	}
	return slices.Contains(perms, permission), nil
}

// GetUserPermissions returns all permissions for a user (Redis-cached when configured).
func (s *RBACService) GetUserPermissions(ctx context.Context, userID uint) ([]string, error) {
	if s.rdb != nil {
		raw, err := s.rdb.Get(ctx, userPermCacheKey(userID)).Bytes()
		if err == nil {
			var cached []string
			if json.Unmarshal(raw, &cached) == nil {
				return cached, nil
			}
		} else if !errors.Is(err, redis.Nil) {
			logger.Warn(ctx, "rbac cache: get", "user_id", userID, "error", err)
		}
	}

	perms, err := s.computeUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	if s.rdb != nil {
		if payload, err := json.Marshal(perms); err == nil {
			if setErr := s.rdb.Set(ctx, userPermCacheKey(userID), payload, rbacPermTTL).Err(); setErr != nil {
				logger.Warn(ctx, "rbac cache: set", "user_id", userID, "error", setErr)
			}
		}
	}

	return perms, nil
}

// GetUserRoles returns all roles for a user with their menus
func (s *RBACService) GetUserRoles(ctx context.Context, userID uint) ([]models.Role, error) {
	isSuperAdmin := s.authSvc != nil && s.authSvc.IsSuperAdmin(userID)
	hasAdminRole, err := s.store.UserHasAdminRole(ctx, userID)
	if err != nil {
		return nil, err
	}

	roles, err := s.store.ListUserRolesWithMenus(ctx, userID)
	if err != nil {
		return nil, err
	}

	if hasAdminRole || isSuperAdmin {
		allMenus, err := s.store.ListVisibleMenus(ctx)
		if err != nil {
			return nil, err
		}
		for i := range roles {
			if roles[i].Code == "admin" || isSuperAdmin {
				roles[i].Menus = allMenus
			}
		}
	}
	return roles, nil
}

// HasAnyPermission checks if user has any of the specified permissions
func (s *RBACService) HasAnyPermission(ctx context.Context, userID uint, permissions []string) (bool, error) {
	if len(permissions) == 0 {
		return false, nil
	}
	if s.authSvc != nil && s.authSvc.IsSuperAdmin(userID) {
		return true, nil
	}
	isAdmin, err := s.store.UserHasAdminRole(ctx, userID)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}
	perms, err := s.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, p := range permissions {
		if slices.Contains(perms, p) {
			return true, nil
		}
	}
	return false, nil
}

// HasAllPermissions checks if user has all of the specified permissions
func (s *RBACService) HasAllPermissions(ctx context.Context, user interface{}, permissions []string) (bool, error) {
	for _, perm := range permissions {
		has, err := s.CheckPermission(ctx, user, perm)
		if err != nil {
			return false, err
		}
		if !has {
			return false, nil
		}
	}
	return true, nil
}
