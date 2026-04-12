package services

import "context"

// PermissionCacheInvalidator clears cached user permission lists when menus, roles, or user–role data changes.
type PermissionCacheInvalidator interface {
	InvalidateUserPermissions(ctx context.Context, userID uint)
	InvalidateAllPermissions(ctx context.Context)
}
