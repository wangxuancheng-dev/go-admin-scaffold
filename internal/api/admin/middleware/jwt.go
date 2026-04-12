package middleware

import (
	"strings"

	"app/internal/core/services"
	"app/pkg/ginext"
	"app/pkg/logger"
	"app/pkg/response"

	"github.com/gin-gonic/gin"
)

// JWT middleware validates JWT tokens
func JWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.UnauthorizedError(c)
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Sugared().Warnw("invalid authorization header format")
			response.UnauthorizedError(c)
			c.Abort()
			return
		}

		// Get auth service and user service
		authSvc, ok := ginext.GetService[*services.AuthService](c, "authService")
		if !ok {
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := authSvc.ValidateToken(tokenString)
		if err != nil {
			logger.Sugared().Warnw("jwt validation failed", "error", err)
			response.UnauthorizedError(c)
			c.Abort()
			return
		}

		// Get user from claims
		user, err := authSvc.GetUserFromClaims(c.Request.Context(), claims)
		if err != nil {
			logger.Sugared().Warnw("get user from jwt claims failed", "error", err)
			response.UnauthorizedError(c)
			c.Abort()
			return
		}

		// Set user in context
		if user == nil {
			logger.Sugared().Warnw("user nil after GetUserFromClaims")
			response.UnauthorizedError(c)
			c.Abort()
			return
		}

		// Set IsSuperAdmin field
		user.IsSuperAdmin = authSvc.IsSuperAdmin(user.ID)

		c.Set("user", user)
		c.Next()
	}
}
