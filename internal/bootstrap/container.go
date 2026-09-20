package bootstrap

import (
	"fmt"

	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/core/repositories"
	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/internal/core/storage"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Container is the application composition root: built once at process start.
type Container struct {
	Config *config.Config
	DB     *gorm.DB
	Redis  *redis.Client

	Log  *services.LogService
	User *services.UserService
	Auth *services.AuthService
	RBAC *services.RBACService
	Role *services.RoleService
	Todo *services.TodoService
	Menu *services.MenuService

	Storage storage.Storage
}

// NewContainer wires repositories and services once from explicit dependencies.
func NewContainer(cfg *config.Config, db *gorm.DB, rdb *redis.Client) (*Container, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}

	userRepo := repositories.NewUserRepository(db)
	userRepo.SetConfig(cfg)
	logRepo := repositories.NewLogRepository(db)
	todoRepo := repositories.NewTodoRepository(db)
	menuRepo := repositories.NewMenuRepository(db)
	roleRepo := repositories.NewRoleRepository(db)
	rbacRepo := repositories.NewRBACRepository(db)

	logSvc := services.NewLogService(logRepo)
	authSvc := services.NewAuthService(userRepo, logSvc, cfg)
	rbacSvc := services.NewRBACService(rbacRepo, authSvc, rdb)
	userSvc := services.NewUserService(userRepo, logSvc, cfg, authSvc, rbacSvc)
	roleSvc := services.NewRoleService(roleRepo, rbacSvc)
	todoSvc := services.NewTodoService(todoRepo)
	menuSvc := services.NewMenuService(menuRepo, userRepo, rbacSvc)

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
	store, err := storage.NewStorage(storageCfg)
	if err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}

	return &Container{
		Config:  cfg,
		DB:      db,
		Redis:   rdb,
		Log:     logSvc,
		User:    userSvc,
		Auth:    authSvc,
		RBAC:    rbacSvc,
		Role:    roleSvc,
		Todo:    todoSvc,
		Menu:    menuSvc,
		Storage: store,
	}, nil
}
