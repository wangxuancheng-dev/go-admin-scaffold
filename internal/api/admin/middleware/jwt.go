package middleware

import (
	"strings"

	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/pkg/logger"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

// JWT validates Bearer tokens using the injected AuthService and ContextLogger.
func JWT(authSvc *services.AuthService, log logger.ContextLogger) gin.HandlerFunc {
	if log == nil {
		log = logger.Default()
	}
	return func(c *gin.Context) {
		if authSvc == nil {
			response.ServerError(c)
			c.Abort()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.UnauthorizedError(c)
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Warn(c.Request.Context(), "invalid authorization header format")
			response.UnauthorizedError(c)
			c.Abort()
			return
		}

		claims, err := authSvc.ValidateToken(parts[1])
		if err != nil {
			log.Warn(c.Request.Context(), "jwt validation failed", "error", err)
			response.UnauthorizedError(c)
			c.Abort()
			return
		}

		user, err := authSvc.GetUserFromClaims(c.Request.Context(), claims)
		if err != nil || user == nil {
			log.Warn(c.Request.Context(), "get user from jwt claims failed", "error", err)
			response.UnauthorizedError(c)
			c.Abort()
			return
		}

		user.IsSuperAdmin = authSvc.IsSuperAdmin(user.ID)
		c.Set("user", user)
		c.Next()
	}
}
