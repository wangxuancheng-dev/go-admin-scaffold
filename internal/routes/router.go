package routes

import (
	"fmt"
	"strings"
	"time"

	"app/internal/api/admin/handlers"
	"app/internal/api/admin/middleware"
	adminv1 "app/internal/api/admin/v1"
	openv1 "app/internal/api/open/v1"
	"app/internal/config"
	corehandlers "app/internal/core/handlers"
	coremiddleware "app/internal/core/middleware"
	"app/internal/core/storage"
	"app/pkg/response"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/time/rate"
)

// responseWriter wraps gin.ResponseWriter to track if response was written
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

// SetupRoutes configures all the routes for the application
func SetupRoutes(r *gin.Engine, cfg *config.Config) error {
	// Global middleware (Trace is applied in cmd/server/setup before routes)
	r.Use(middleware.I18n())                // Add i18n middleware globally
	r.Use(middleware.ServiceInjection(cfg)) // Add service injection middleware globally

	// Swagger and dev-only test routes (avoid exposing in production)
	if !strings.EqualFold(cfg.App.Env, "production") {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

		test := r.Group("/api/test")
		{
			testHandler := corehandlers.NewTestHandler()
			test.GET("/ratelimit", coremiddleware.RateLimit(0.2, 2), testHandler.RateLimitTest)
			test.GET("/ratelimit2", coremiddleware.RateLimit(5, 10), testHandler.RateLimitTest)
		}
	}

	// Serve static files
	r.Static("/static", "./static")

	// Admin API routes (v1)
	adminV1 := r.Group("/api/admin/v1")
	{
		// Auth routes (no JWT protection needed)
		auth := adminV1.Group("/auth")
		{
			auth.GET("/captcha", coremiddleware.RateLimit(rate.Every(200*time.Millisecond), 40), wrapHandler(adminv1.GetCaptcha))
			auth.POST("/login", coremiddleware.RateLimit(rate.Every(time.Second), 8), wrapHandler(adminv1.Login))
			auth.POST("/logout", middleware.JWT(), wrapHandler(adminv1.Logout))
			auth.POST("/refresh", middleware.JWT(), coremiddleware.RateLimit(rate.Every(200*time.Millisecond), 30), wrapHandler(adminv1.RefreshToken))
		}

		// WebSocket routes (no JWT middleware needed, token passed via query params)
		wsHandler := handlers.NewWSHandler()
		adminV1.GET("/ws", wrapHandler(wsHandler.HandleWebSocket))
		adminV1.POST("/ws/join", middleware.JWT(), wrapHandler(wsHandler.JoinGroup))
		adminV1.POST("/ws/leave", middleware.JWT(), wrapHandler(wsHandler.LeaveGroup))
		adminV1.POST("/ws/send", middleware.JWT(), wrapHandler(wsHandler.SendMessage))

		// SSE routes
		sseHandler := handlers.NewSSEHandler()
		adminV1.GET("/sse", wrapHandler(sseHandler.HandleSSE))
		adminV1.POST("/sse/notify", middleware.JWT(), wrapHandler(sseHandler.SendNotification))
		adminV1.POST("/sse/join", middleware.JWT(), wrapHandler(sseHandler.JoinGroup))
		adminV1.POST("/sse/leave", middleware.JWT(), wrapHandler(sseHandler.LeaveGroup))
	}

	// Protected Admin API routes
	adminV1Protected := r.Group("/api/admin/v1")
	adminV1Protected.Use(middleware.JWT())          // Protect all admin routes with JWT auth
	adminV1Protected.Use(middleware.OperationLog()) // Add operation logging
	{
		// User routes
		users := adminV1Protected.Group("/users")
		{
			users.GET("", middleware.RBAC("user:view"), wrapHandler(adminv1.ListUsers))
			users.POST("", middleware.RBAC("user:create"), wrapHandler(adminv1.CreateUser))
			users.GET("/:id", middleware.RBAC("user:view"), wrapHandler(adminv1.GetUser))
			users.PUT("/:id", middleware.RBAC("user:edit"), wrapHandler(adminv1.UpdateUser))
			users.DELETE("/:id", middleware.RBAC("user:delete"), wrapHandler(adminv1.DeleteUser))
			users.PUT("/:id/status", middleware.RBAC("user:edit"), wrapHandler(adminv1.UpdateUserStatus))
			users.GET("/:id/logs", middleware.RBAC("log:view"), wrapHandler(adminv1.GetUserLogs))
			users.PUT("/:id/roles", middleware.RBAC("user:edit"), wrapHandler(adminv1.UpdateUserRoles))
		}

		// Role routes
		roles := adminV1Protected.Group("/roles")
		{
			roles.GET("", middleware.RBAC("role:view"), wrapHandler(adminv1.ListRoles))
			roles.POST("", middleware.RBAC("role:create"), wrapHandler(adminv1.CreateRole))
			roles.GET("/:id", middleware.RBAC("role:view"), wrapHandler(adminv1.GetRole))
			roles.PUT("/:id", middleware.RBAC("role:edit"), wrapHandler(adminv1.UpdateRole))
			roles.DELETE("/:id", middleware.RBAC("role:delete"), wrapHandler(adminv1.DeleteRole))
			roles.GET("/:id/menus", middleware.RBAC("role:view"), wrapHandler(adminv1.GetRoleMenus))
			roles.PUT("/:id/menus", middleware.RBAC("role:edit"), wrapHandler(adminv1.UpdateRoleMenus))
		}

		// Menu routes
		menus := adminV1Protected.Group("/menus")
		{
			menus.GET("", middleware.RBAC("menu:view"), wrapHandler(adminv1.ListMenus))
			menus.POST("", middleware.RBAC("menu:create"), wrapHandler(adminv1.CreateMenu))
			menus.GET("/tree", middleware.RBAC("menu:view"), wrapHandler(adminv1.GetMenuTree))
			menus.GET("/user", wrapHandler(adminv1.GetUserMenus)) // No permission check as it's user's own menus
			menus.GET("/:id", middleware.RBAC("menu:view"), wrapHandler(adminv1.GetMenu))
			menus.PUT("/:id", middleware.RBAC("menu:edit"), wrapHandler(adminv1.UpdateMenu))
			menus.DELETE("/:id", middleware.RBAC("menu:delete"), wrapHandler(adminv1.DeleteMenu))
			menus.PUT("/:id/roles", middleware.RBAC("menu:edit"), wrapHandler(adminv1.UpdateMenuRoles))
		}

		// Log routes
		logs := adminV1Protected.Group("/logs")
		logs.Use(middleware.RBAC("log:view"))
		{
			logs.GET("/login", wrapHandler(adminv1.ListLoginLogs))
			logs.GET("/operation", wrapHandler(adminv1.ListOperationLogs))
		}

		// I18n routes
		i18n := adminV1Protected.Group("/i18n")
		{
			i18n.GET("/locales", wrapHandler(adminv1.GetLocales))
			i18n.GET("/translations", wrapHandler(adminv1.GetTranslations))
		}

		// Dashboard routes (accessible to all authenticated users)
		dashboard := adminV1Protected.Group("/dashboard")
		dashboard.Use(middleware.RBAC("dashboard:view"))
		{
			// Add dashboard endpoints here when needed
		}

		// Profile routes (accessible to all authenticated users)
		profile := adminV1Protected.Group("/profile")
		{
			profile.GET("", wrapHandler(adminv1.GetCurrentUser))
			profile.PUT("", wrapHandler(adminv1.UpdateCurrentUser))
		}

		// 创建存储实例
		storageCfg := &storage.Config{
			Driver:    cfg.Storage.Driver,
			LocalPath: cfg.Storage.Local.Path,
			S3Config: &storage.S3Config{
				Endpoint:        cfg.Storage.S3.Endpoint,
				AccessKeyID:     cfg.Storage.S3.AccessKeyID,
				SecretAccessKey: cfg.Storage.S3.SecretAccessKey,
				Bucket:          cfg.Storage.S3.Bucket,
				Region:          cfg.Storage.S3.Region,
				UseSSL:          cfg.Storage.S3.UseSSL,
			},
		}

		uploadBackend, err := storage.NewStorage(storageCfg)
		if err != nil {
			return fmt.Errorf("init storage: %w", err)
		}

		// 创建上传处理器
		uploadHandler := corehandlers.NewUploadHandler(uploadBackend)

		// 上传相关路由
		upload := adminV1Protected.Group("/upload")
		upload.Use(middleware.RBAC("upload:create"))
		{
			upload.POST("/file", wrapHandler(uploadHandler.Upload))       // 单文件上传
			upload.POST("/files", wrapHandler(uploadHandler.MultiUpload)) // 多文件上传
		}

		// Todo routes
		todos := adminV1Protected.Group("/todos")
		{
			todos.GET("", middleware.RBAC("todo:view"), wrapHandler(adminv1.ListTodos))
			todos.POST("", middleware.RBAC("todo:create"), wrapHandler(adminv1.CreateTodo))
			todos.GET("/:id", middleware.RBAC("todo:view"), wrapHandler(adminv1.GetTodo))
			todos.PUT("/:id", middleware.RBAC("todo:edit"), wrapHandler(adminv1.UpdateTodo))
			todos.DELETE("/:id", middleware.RBAC("todo:delete"), wrapHandler(adminv1.DeleteTodo))
		}

	}

	// Open API routes (v1)
	openV1 := r.Group("/api/open/v1")
	{
		// Public routes
		public := openV1.Group("/public")
		{
			public.GET("/health", wrapHandler(openv1.HealthCheck))
			public.GET("/live", wrapHandler(openv1.Liveness))
			public.GET("/ready", wrapHandler(openv1.Readiness))
		}

		// OAuth routes
		oauth := openV1.Group("/oauth")
		{
			oauth.GET("/github", wrapHandler(openv1.GithubOAuth))
			oauth.GET("/github/callback", wrapHandler(openv1.GithubOAuthCallback))
		}
	}

	return nil
}

// wrapHandler wraps a gin.HandlerFunc to ensure consistent response handling
func wrapHandler(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Store the original writer
		originalWriter := c.Writer

		// Create a custom response writer that captures the response
		responseWriter := &responseWriter{ResponseWriter: originalWriter}
		c.Writer = responseWriter

		// Call the original handler
		handler(c)

		// If no response has been written and it's not a redirect
		if !responseWriter.written && c.Writer.Status() < 300 {
			// Ensure response includes trace_id
			response.Success(c, nil)
		}
	}
}
