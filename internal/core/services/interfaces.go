package services

import (
	"context"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/types"
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

// RoleServiceAPI is the subset of role operations used by admin HTTP handlers.
type RoleServiceAPI interface {
	List(ctx context.Context, pagination *models.Pagination) ([]models.Role, error)
	Create(ctx context.Context, req *CreateRoleRequest) (*models.Role, error)
	GetByID(ctx context.Context, id uint) (*models.Role, error)
	Update(ctx context.Context, id uint, req *UpdateRoleRequest) (*models.Role, error)
	Delete(ctx context.Context, id uint) error
	GetMenus(ctx context.Context, roleID uint) ([]models.Menu, error)
	UpdateMenus(ctx context.Context, roleID uint, req *UpdateRoleMenusRequest) error
}

// MenuServiceAPI is the subset of menu operations used by admin HTTP handlers.
type MenuServiceAPI interface {
	GetAll(ctx context.Context) ([]models.Menu, error)
	GetTree(ctx context.Context) ([]models.Menu, error)
	GetUserMenus(ctx context.Context, userID uint) ([]MenuRouteItem, error)
	Create(ctx context.Context, req *CreateMenuRequest) (*models.Menu, error)
	GetByID(ctx context.Context, id uint) (*models.Menu, error)
	Update(ctx context.Context, id uint, req *UpdateMenuRequest) (*models.Menu, error)
	Delete(ctx context.Context, id uint) error
	UpdateMenuRoles(ctx context.Context, menuID uint, roleIDs []uint) error
}

// UserRepository defines the interface for user data access
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id uint) (*models.User, error)
	FindBasicByID(ctx context.Context, id uint) (*models.User, error)
	ListWithRoles(ctx context.Context, pagination *models.Pagination) ([]models.User, error)
	ListWithFilters(ctx context.Context, pagination *models.Pagination, filters *types.UserSearchFilters) ([]models.User, error)
	ExportWithFilters(ctx context.Context, filters *types.UserExportFilters) ([]models.User, error)
	Create(ctx context.Context, user *models.User) error
	CreateWithRoles(ctx context.Context, user *models.User, roleIDs []uint) error
	Update(ctx context.Context, user *models.User) error
	UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error
	UpdateStatus(ctx context.Context, id uint, status int) error
	ReplaceUserRoles(ctx context.Context, userID uint, roleIDs []uint) error
	Delete(ctx context.Context, id uint) error
	UpdateLastLogin(ctx context.Context, userID uint) error
}

// RoleRepository defines the interface for role data access
type RoleRepository interface {
	ListWithMenus(ctx context.Context, pagination *models.Pagination) ([]models.Role, error)
	FindByIDWithMenus(ctx context.Context, id uint) (*models.Role, error)
	CreateWithMenus(ctx context.Context, role *models.Role, menuIDs []uint) error
	UpdateWithMenus(ctx context.Context, id uint, role *models.Role, menuIDs *[]uint) (*models.Role, error)
	DeleteWithAssociations(ctx context.Context, id uint) error
	ListMenusByRoleID(ctx context.Context, roleID uint) ([]models.Menu, error)
	SetMenus(ctx context.Context, roleID uint, menuIDs []uint) error
}

// TodoRepository defines the interface for todo data access
type TodoRepository interface {
	Create(ctx context.Context, todo *models.Todo) error
	List(ctx context.Context, pagination *models.Pagination) ([]models.Todo, error)
	GetByID(ctx context.Context, id uint) (*models.Todo, error)
	Update(ctx context.Context, todo *models.Todo) error
	Delete(ctx context.Context, id uint) error
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

// MenuRepository defines the interface for menu data access
type MenuRepository interface {
	FindByID(ctx context.Context, id uint) (*models.Menu, error)
	FindAll(ctx context.Context) ([]models.Menu, error)
	FindByParentID(ctx context.Context, parentID *uint) ([]models.Menu, error)
	FindTree(ctx context.Context) ([]models.Menu, error)
	FindByRoleIDs(ctx context.Context, roleIDs []uint) ([]models.Menu, error)
	FindVisibleMenus(ctx context.Context) ([]models.Menu, error)
	Create(ctx context.Context, menu *models.Menu) error
	Update(ctx context.Context, menu *models.Menu) error
	Delete(ctx context.Context, id uint) error
	UpdateMenuRoles(ctx context.Context, menuID uint, roleIDs []uint) error
	GetMaxSort(ctx context.Context, parentID *uint) (int, error)
}
