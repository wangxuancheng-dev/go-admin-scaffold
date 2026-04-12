package setup

import (
	"app/internal/api/admin/middleware"
	"app/internal/bootstrap"
	"app/internal/config"
	"app/internal/routes"
	"app/pkg/i18n"
	"app/pkg/logger"

	"github.com/gin-gonic/gin"
)

type App struct {
	engine *gin.Engine
	config *config.Config
}

// Engine returns the Gin engine
func (a *App) Engine() *gin.Engine {
	return a.engine
}

func InitializeApp() (*App, error) {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Initialize logger
	if err := logger.Setup(&logger.Config{
		Level:      cfg.Log.Level,
		Filename:   cfg.Log.Filename,
		MaxSize:    cfg.Log.MaxSize,
		MaxAge:     cfg.Log.MaxAge,
		MaxBackups: cfg.Log.MaxBackups,
		Compress:   cfg.Log.Compress,
		Daily:      cfg.Log.Daily,
		Timezone:   cfg.Log.Timezone,
	}); err != nil {
		return nil, err
	}

	// Initialize database
	if err := bootstrap.SetupDatabase(cfg); err != nil {
		return nil, err
	}

	// Initialize Redis
	if err := bootstrap.SetupRedis(cfg); err != nil {
		return nil, err
	}

	// Initialize Cache
	if err := bootstrap.SetupCache(cfg); err != nil {
		return nil, err
	}

	// Initialize i18n
	if err := i18n.Init(&cfg.I18n); err != nil {
		return nil, err
	}

	// Create Gin engine
	engine := gin.New()

	// Use trace middleware first to ensure trace ID is available for all other middleware
	engine.Use(middleware.Trace())

	// Use logger middleware
	engine.Use(gin.Logger())

	// Use custom recovery middleware instead of gin.Recovery()
	engine.Use(middleware.Recovery())

	// Use CORS middleware
	engine.Use(middleware.CORS(&cfg.CORS))

	// Setup routes
	if err := routes.SetupRoutes(engine, cfg); err != nil {
		return nil, err
	}

	return &App{
		engine: engine,
		config: cfg,
	}, nil
}

func (a *App) Run() error {
	return a.engine.Run(a.config.Server.Address)
}

// SetupApp configures and sets up the application
func SetupApp(r *gin.Engine) error {
	// Setup logger
	if err := setupLogger(); err != nil {
		return err
	}

	// Register global middleware
	setupMiddleware(r)

	return nil
}

// setupLogger configures the logger
func setupLogger() error {
	config := &logger.Config{
		Level:      "debug",
		Filename:   "storage/logs/app.log",
		MaxSize:    100,  // 100MB
		MaxBackups: 10,   // Keep 10 old files
		MaxAge:     30,   // 30 days
		Compress:   true, // Compress old files
		Daily:      true, // Rotate daily
		Timezone:   "",   // empty = time.Local
	}

	return logger.Setup(config)
}

// setupMiddleware registers global middleware
func setupMiddleware(r *gin.Engine) {
	// Add trace middleware first to ensure trace ID is available
	r.Use(middleware.Trace())

	// Add other middleware
	r.Use(middleware.Recovery())
	// ... other middleware
}
