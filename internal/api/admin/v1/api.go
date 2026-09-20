package v1

import (
	"go-admin-scaffold/internal/bootstrap"
	"go-admin-scaffold/internal/core/services"
)

// AdminAPI groups constructor-injected admin HTTP handlers.
type AdminAPI struct {
	Auth     *AuthHandler
	Users    *UserHandler
	Roles    *RoleHandler
	Menus    *MenuHandler
	Logs     *LogHandler
	Todos    *TodoHandler
	Profile  *ProfileHandler
	I18n     *I18nHandler
	Realtime *RealtimeHandler
}

// NewAdminAPI builds handlers from the composition-root container.
func NewAdminAPI(c *bootstrap.Container) *AdminAPI {
	return &AdminAPI{
		Auth:     NewAuthHandler(c.Auth),
		Users:    NewUserHandler(c.User, c.Log),
		Roles:    NewRoleHandler(c.Role, c.Menu),
		Menus:    NewMenuHandler(c.Menu),
		Logs:     NewLogHandler(c.Log),
		Todos:    NewTodoHandler(c.Todo),
		Profile:  NewProfileHandler(c.User, c.RBAC),
		I18n:     NewI18nHandler(),
		Realtime: NewRealtimeHandler(c.Realtime),
	}
}

// Ensure interfaces used by handlers stay intentional.
var (
	_ services.UserServiceAPI = (*services.UserService)(nil)
	_ services.RoleServiceAPI = (*services.RoleService)(nil)
	_ services.MenuServiceAPI = (*services.MenuService)(nil)
)
