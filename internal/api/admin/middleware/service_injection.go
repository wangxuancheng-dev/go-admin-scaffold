package middleware

import (
	"app/internal/config"
	"app/internal/core/repositories"
	"app/internal/core/services"
	"app/pkg/database"
	"app/pkg/redis"

	"github.com/gin-gonic/gin"
)

// ServiceInjection injects all required services into the gin context
func ServiceInjection(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := database.GetDB()

		// Initialize repositories
		userRepo := repositories.NewUserRepository(db)
		logRepo := repositories.NewLogRepository(db)
		todoRepo := repositories.NewTodoRepository(db)
		menuRepo := repositories.NewMenuRepository(db)

		// Set config in userRepo
		userRepo.SetConfig(cfg)

		// Initialize services
		logSvc := services.NewLogService(logRepo)
		userSvc := services.NewUserService(userRepo, logSvc, cfg)
		authSvc := services.NewAuthService(userRepo, logSvc, cfg)
		rbacSvc := services.NewRBACService(db)
		rbacSvc.SetRedisClient(redis.GetClient())
		roleSvc := services.NewRoleService(db)
		todoService := services.NewTodoService(todoRepo)
		menuSvc := services.NewMenuService(menuRepo, userRepo)

		// Set up service dependencies
		userSvc.SetAuthService(authSvc)
		rbacSvc.SetAuthService(authSvc)
		menuSvc.SetPermissionCacheInvalidator(rbacSvc)
		roleSvc.SetPermissionCacheInvalidator(rbacSvc)
		userSvc.SetPermissionCacheInvalidator(rbacSvc)

		// Inject services into context
		c.Set("logService", logSvc)
		c.Set("userService", userSvc)
		c.Set("authService", authSvc)
		c.Set("rbacService", rbacSvc)
		c.Set("roleService", roleSvc)
		c.Set("todoService", todoService)
		c.Set("menuService", menuSvc)

		c.Next()
	}
}
