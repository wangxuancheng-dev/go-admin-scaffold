package routes

import (
	"strings"
	"time"

	"go-admin-scaffold/internal/api/admin/handlers"
	"go-admin-scaffold/internal/api/admin/middleware"
	adminv1 "go-admin-scaffold/internal/api/admin/v1"
	openv1 "go-admin-scaffold/internal/api/open/v1"
	"go-admin-scaffold/internal/bootstrap"
	corehandlers "go-admin-scaffold/internal/core/handlers"
	coremiddleware "go-admin-scaffold/internal/core/middleware"
	"go-admin-scaffold/internal/core/metrics"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type responseWriter struct {
	gin.ResponseWriter
	written bool
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.written = true
	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.written = true
	w.ResponseWriter.WriteHeader(statusCode)
}

// SetupRoutes configures all routes using the process-scoped dependency container.
func SetupRoutes(r *gin.Engine, c *bootstrap.Container) error {
	cfg := c.Config
	api := adminv1.NewAdminAPI(c)
	jwt := middleware.JWT(c.Auth)
	opLog := middleware.OperationLog(c.Log)
	rbac := func(perm string) gin.HandlerFunc { return middleware.RBAC(c.RBAC, perm) }

	rdb := c.Redis
	limitLogin := coremiddleware.RateLimitRedis(rdb, time.Second, 8)
	limitCaptcha := coremiddleware.RateLimitRedis(rdb, 200*time.Millisecond, 40)
	limitRefresh := coremiddleware.RateLimitRedis(rdb, 200*time.Millisecond, 30)

	r.Use(middleware.I18n())
	r.Use(metrics.Middleware())
	r.GET("/metrics", metrics.Handler)

	health := openv1.NewHealthHandler(c.DB, c.Redis)

	if !strings.EqualFold(cfg.App.Env, "production") {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		test := r.Group("/api/test")
		{
			testHandler := corehandlers.NewTestHandler()
			test.GET("/ratelimit", coremiddleware.RateLimit(0.2, 2), testHandler.RateLimitTest)
			test.GET("/ratelimit2", coremiddleware.RateLimit(5, 10), testHandler.RateLimitTest)
		}
	}

	r.Static("/static", "./static")
	if strings.EqualFold(cfg.Storage.Driver, "local") || cfg.Storage.Driver == "" {
		uploadPath := cfg.Storage.Local.Path
		if uploadPath == "" {
			uploadPath = "storage/uploads"
		}
		r.Static("/uploads", uploadPath)
	}

	wsHandler := handlers.NewWSHandler(c.Auth)
	sseHandler := handlers.NewSSEHandler(c.Auth)
	uploadHandler := corehandlers.NewUploadHandler(c.Storage)

	adminV1 := r.Group("/api/admin/v1")
	{
		auth := adminV1.Group("/auth")
		{
			auth.GET("/captcha", limitCaptcha, wrapHandler(api.Auth.GetCaptcha))
			auth.POST("/login", limitLogin, wrapHandler(api.Auth.Login))
			auth.POST("/logout", jwt, wrapHandler(api.Auth.Logout))
			auth.POST("/refresh", jwt, limitRefresh, wrapHandler(api.Auth.RefreshToken))
		}

		adminV1.GET("/ws", wrapHandler(wsHandler.HandleWebSocket))
		adminV1.POST("/ws/join", jwt, wrapHandler(wsHandler.JoinGroup))
		adminV1.POST("/ws/leave", jwt, wrapHandler(wsHandler.LeaveGroup))
		adminV1.POST("/ws/send", jwt, wrapHandler(wsHandler.SendMessage))

		adminV1.GET("/sse", wrapHandler(sseHandler.HandleSSE))
		adminV1.POST("/sse/notify", jwt, wrapHandler(sseHandler.SendNotification))
		adminV1.POST("/sse/join", jwt, wrapHandler(sseHandler.JoinGroup))
		adminV1.POST("/sse/leave", jwt, wrapHandler(sseHandler.LeaveGroup))
	}

	adminV1Protected := r.Group("/api/admin/v1")
	adminV1Protected.Use(jwt)
	adminV1Protected.Use(opLog)
	{
		users := adminV1Protected.Group("/users")
		{
			users.GET("", rbac("user:view"), wrapHandler(api.Users.ListUsers))
			users.POST("", rbac("user:create"), wrapHandler(api.Users.CreateUser))
			users.GET("/:id", rbac("user:view"), wrapHandler(api.Users.GetUser))
			users.PUT("/:id", rbac("user:edit"), wrapHandler(api.Users.UpdateUser))
			users.DELETE("/:id", rbac("user:delete"), wrapHandler(api.Users.DeleteUser))
			users.PUT("/:id/status", rbac("user:edit"), wrapHandler(api.Users.UpdateUserStatus))
			users.GET("/:id/logs", rbac("log:view"), wrapHandler(api.Users.GetUserLogs))
			users.PUT("/:id/roles", rbac("user:edit"), wrapHandler(api.Users.UpdateUserRoles))
		}

		roles := adminV1Protected.Group("/roles")
		{
			roles.GET("", rbac("role:view"), wrapHandler(api.Roles.ListRoles))
			roles.POST("", rbac("role:create"), wrapHandler(api.Roles.CreateRole))
			roles.GET("/:id", rbac("role:view"), wrapHandler(api.Roles.GetRole))
			roles.PUT("/:id", rbac("role:edit"), wrapHandler(api.Roles.UpdateRole))
			roles.DELETE("/:id", rbac("role:delete"), wrapHandler(api.Roles.DeleteRole))
			roles.GET("/:id/menus", rbac("role:view"), wrapHandler(api.Roles.GetRoleMenus))
			roles.PUT("/:id/menus", rbac("role:edit"), wrapHandler(api.Roles.UpdateRoleMenus))
		}

		menus := adminV1Protected.Group("/menus")
		{
			menus.GET("", rbac("menu:view"), wrapHandler(api.Menus.ListMenus))
			menus.POST("", rbac("menu:create"), wrapHandler(api.Menus.CreateMenu))
			menus.GET("/tree", rbac("menu:view"), wrapHandler(api.Menus.GetMenuTree))
			menus.GET("/user", wrapHandler(api.Menus.GetUserMenus))
			menus.GET("/:id", rbac("menu:view"), wrapHandler(api.Menus.GetMenu))
			menus.PUT("/:id", rbac("menu:edit"), wrapHandler(api.Menus.UpdateMenu))
			menus.DELETE("/:id", rbac("menu:delete"), wrapHandler(api.Menus.DeleteMenu))
			menus.PUT("/:id/roles", rbac("menu:edit"), wrapHandler(api.Menus.UpdateMenuRoles))
		}

		logs := adminV1Protected.Group("/logs")
		logs.Use(rbac("log:view"))
		{
			logs.GET("/login", wrapHandler(api.Logs.ListLoginLogs))
			logs.GET("/operation", wrapHandler(api.Logs.ListOperationLogs))
		}

		i18n := adminV1Protected.Group("/i18n")
		{
			i18n.GET("/locales", wrapHandler(api.I18n.GetLocales))
			i18n.GET("/translations", wrapHandler(api.I18n.GetTranslations))
		}

		dashboard := adminV1Protected.Group("/dashboard")
		dashboard.Use(rbac("dashboard:view"))
		_ = dashboard

		profile := adminV1Protected.Group("/profile")
		{
			profile.GET("", wrapHandler(api.Profile.GetCurrentUser))
			profile.PUT("", wrapHandler(api.Profile.UpdateCurrentUser))
		}

		upload := adminV1Protected.Group("/upload")
		upload.Use(rbac("upload:create"))
		{
			upload.POST("/file", wrapHandler(uploadHandler.Upload))
			upload.POST("/files", wrapHandler(uploadHandler.MultiUpload))
		}

		todos := adminV1Protected.Group("/todos")
		{
			todos.GET("", rbac("todo:view"), wrapHandler(api.Todos.ListTodos))
			todos.POST("", rbac("todo:create"), wrapHandler(api.Todos.CreateTodo))
			todos.GET("/:id", rbac("todo:view"), wrapHandler(api.Todos.GetTodo))
			todos.PUT("/:id", rbac("todo:edit"), wrapHandler(api.Todos.UpdateTodo))
			todos.DELETE("/:id", rbac("todo:delete"), wrapHandler(api.Todos.DeleteTodo))
		}
	}

	openV1 := r.Group("/api/open/v1")
	{
		public := openV1.Group("/public")
		{
			public.GET("/health", wrapHandler(health.HealthCheck))
			public.GET("/live", wrapHandler(health.Liveness))
			public.GET("/ready", wrapHandler(health.Readiness))
		}
		oauth := openV1.Group("/oauth")
		{
			oauth.GET("/github", wrapHandler(openv1.GithubOAuth))
			oauth.GET("/github/callback", wrapHandler(openv1.GithubOAuthCallback))
		}
	}

	return nil
}

func wrapHandler(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		originalWriter := c.Writer
		rw := &responseWriter{ResponseWriter: originalWriter}
		c.Writer = rw
		handler(c)
		if !rw.written && c.Writer.Status() < 300 {
			response.Success(c, nil)
		}
	}
}
