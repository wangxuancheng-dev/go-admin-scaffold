package setup

import (
	"go-admin-scaffold/internal/api/admin/middleware"
	"go-admin-scaffold/internal/bootstrap"
	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/routes"
	"go-admin-scaffold/pkg/i18n"
	"go-admin-scaffold/pkg/logger"

	"github.com/gin-gonic/gin"
)

type App struct {
	engine    *gin.Engine
	config    *config.Config
	container *bootstrap.Container
}

func (a *App) Engine() *gin.Engine {
	return a.engine
}

func (a *App) Container() *bootstrap.Container {
	return a.container
}

func InitializeApp() (*App, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

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

	db, err := bootstrap.SetupDatabase(cfg)
	if err != nil {
		return nil, err
	}
	rdb, err := bootstrap.SetupRedis(cfg)
	if err != nil {
		return nil, err
	}
	cch, err := bootstrap.SetupCache(cfg, rdb)
	if err != nil {
		return nil, err
	}
	if err := i18n.Init(&cfg.I18n); err != nil {
		return nil, err
	}

	container, err := bootstrap.NewContainerWithCache(cfg, db, rdb, cch)
	if err != nil {
		return nil, err
	}

	engine := gin.New()
	engine.Use(middleware.Trace())
	engine.Use(gin.Logger())
	engine.Use(middleware.Recovery())
	engine.Use(middleware.CORS(&cfg.CORS))

	if err := routes.SetupRoutes(engine, container); err != nil {
		return nil, err
	}

	return &App{
		engine:    engine,
		config:    cfg,
		container: container,
	}, nil
}

func (a *App) Run() error {
	return a.engine.Run(a.config.Server.Address)
}
