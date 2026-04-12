package middleware

import (
	"app/internal/core/services"
	"app/pkg/ginext"
	"app/pkg/response"

	"github.com/gin-gonic/gin"
)

// RBAC middleware checks if the user has the required permissions
func RBAC(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		user, exists := c.Get("user")
		if !exists {
			response.UnauthorizedError(c)
			c.Abort()
			return
		}

		rbacSvc, ok := ginext.GetService[*services.RBACService](c, "rbacService")
		if !ok {
			return
		}

		// Check if user has permission
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
