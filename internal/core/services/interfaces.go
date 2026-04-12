package services

import (
	"context"

	"app/internal/core/models"
	"app/internal/core/types"

	"gorm.io/gorm"
)

// UserServiceAPI is the subset of user operations used by admin HTTP handlers.
// *UserService implements this interface; tests may provide mocks.
type UserServiceAPI interface {
	ListWithFilters(ctx context.Context, pagination *models.Pagination, filters *types.UserSearchFilters) ([]models.User, error)
	Create(ctx context.Context, req *CreateUserRequest) (*models.User, error)
	GetByID(ctx context.Context, id uint) (*models.User, error)
	Update(ctx context.Context, id uint, req *UpdateUserRequest) (*models.User, error)
	Delete(ctx context.Context, id uint) error
	ExportUserList(ctx context.Context, req *ExportUserListRequest) ([]models.User, error)
	UpdateUserRoles(ctx context.Context, userID uint, roleIDs []uint) error
	UpdateStatus(ctx context.Context, id uint, status int) error
	IsSuperAdmin(userID uint) bool
}

// UserRepository defines the interface for user data access
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id uint) (*models.User, error)
	ListWithRoles(ctx context.Context, pagination *models.Pagination) ([]models.User, error)
	ListWithFilters(ctx context.Context, pagination *models.Pagination, filters *types.UserSearchFilters) ([]models.User, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uint) error
	UpdateLastLogin(ctx context.Context, userID uint) error
	GetDB() *gorm.DB
}

// LogRepository defines the interface for log data access
type LogRepository interface {
	CreateLoginLog(ctx context.Context, log *models.LoginLog) error
	CreateOperationLog(ctx context.Context, log *models.OperationLog) error
	ListLoginLogs(ctx context.Context, pagination *models.Pagination, query map[string]interface{}) ([]models.LoginLog, error)
	ListOperationLogs(ctx context.Context, pagination *models.Pagination, query map[string]interface{}) ([]models.OperationLog, error)
	GetLoginLogsByUserID(ctx context.Context, userID uint, limit int) ([]models.LoginLog, error)
	GetOperationLogsByUserID(ctx context.Context, userID uint, limit int) ([]models.OperationLog, error)
}
