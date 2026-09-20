package middleware

import (
	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

// RBAC checks whether the authenticated user has the required permission.
func RBAC(rbacSvc *services.RBACService, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			response.UnauthorizedError(c)
			c.Abort()
			return
		}
		if rbacSvc == nil {
			response.ServerError(c)
			c.Abort()
			return
		}

		hasPermission, err := rbacSvc.CheckPermission(c.Request.Context(), user, permission)
		if err != nil {
			response.ServerError(c)
			c.Abort()
			return
		}
		if !hasPermission {
			response.ForbiddenError(c)
			c.Abort()
			return
		}
		c.Next()
	}
}
