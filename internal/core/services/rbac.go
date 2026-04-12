package services

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"app/internal/core/models"
	"app/pkg/logger"
)

const (
	rbacPermKeyPrefix = "rbac:perm:"
	rbacPermTTL       = 5 * time.Minute
)

// RBACService handles role-based access control
type RBACService struct {
	db      *gorm.DB
	authSvc AuthServiceInterface
	rdb     *redis.Client
}

// NewRBACService creates a new RBAC service instance
func NewRBACService(db *gorm.DB) *RBACService {
	return &RBACService{
		db: db,
	}
}

// SetAuthService sets the auth service instance
func (s *RBACService) SetAuthService(authSvc AuthServiceInterface) {
	s.authSvc = authSvc
}

// SetRedisClient sets the Redis client used for permission list caching (optional).
func (s *RBACService) SetRedisClient(c *redis.Client) {
	s.rdb = c
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
		var allPermissions []string
		err := s.db.WithContext(ctx).Model(&models.Menu{}).
			Where("status = 1 AND visible = 1 AND permission != ''").
			Pluck("permission", &allPermissions).Error
		return allPermissions, err
	}

	var isAdmin int64
	err := s.db.WithContext(ctx).Table("user_roles").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.code = 'admin' AND roles.status = 1", userID).
		Count(&isAdmin).Error
	if err != nil {
		return nil, err
	}

	if isAdmin > 0 {
		var allPermissions []string
		err := s.db.WithContext(ctx).Model(&models.Menu{}).
			Where("status = 1 AND visible = 1 AND permission != ''").
			Pluck("permission", &allPermissions).Error
		return allPermissions, err
	}

	var permissions []string
	err = s.db.WithContext(ctx).Table("user_roles").
		Select("DISTINCT menus.permission").
		Joins("JOIN role_menus ON user_roles.role_id = role_menus.role_id").
		Joins("JOIN menus ON role_menus.menu_id = menus.id").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND menus.status = 1 AND menus.visible = 1 AND menus.permission != '' AND roles.status = 1", userID).
		Pluck("menus.permission", &permissions).Error

	if err != nil {
		return nil, err
	}

	return permissions, nil
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

	var count int64
	err := s.db.WithContext(ctx).Table("user_roles").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.code = 'admin' AND roles.status = 1", userModel.ID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	if count > 0 {
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
	var roles []models.Role

	isSuperAdmin := s.authSvc != nil && s.authSvc.IsSuperAdmin(userID)

	var hasAdminRole bool
	err := s.db.WithContext(ctx).Table("user_roles").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.code = 'admin' AND roles.status = 1", userID).
		Limit(1).Find(&roles).Error
	if err != nil {
		return nil, err
	}
	hasAdminRole = len(roles) > 0

	if hasAdminRole || isSuperAdmin {
		err = s.db.WithContext(ctx).
			Preload("Menus", "status = 1 AND visible = 1").
			Joins("JOIN user_roles ON roles.id = user_roles.role_id").
			Where("user_roles.user_id = ? AND roles.status = 1", userID).
			Find(&roles).Error

		for i := range roles {
			if roles[i].Code == "admin" || isSuperAdmin {
				var allMenus []models.Menu
				if err := s.db.WithContext(ctx).Where("status = 1 AND visible = 1").Find(&allMenus).Error; err != nil {
					return nil, err
				}
				roles[i].Menus = allMenus
			}
		}
	} else {
		err = s.db.WithContext(ctx).
			Preload("Menus", "status = 1 AND visible = 1").
			Joins("JOIN user_roles ON roles.id = user_roles.role_id").
			Where("user_roles.user_id = ? AND roles.status = 1", userID).
			Find(&roles).Error
	}

	if err != nil {
		return nil, err
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

	var count int64
	err := s.db.WithContext(ctx).Table("user_roles").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.code = 'admin' AND roles.status = 1", userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	if count > 0 {
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
