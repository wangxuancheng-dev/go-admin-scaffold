package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-admin-scaffold/internal/api/admin/middleware"
	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/internal/core/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type jwtUserRepo struct {
	user *models.User
}

func (r *jwtUserRepo) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *jwtUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *jwtUserRepo) FindByID(ctx context.Context, id uint) (*models.User, error) {
	if r.user != nil && r.user.ID == id {
		return r.user, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *jwtUserRepo) ListWithRoles(ctx context.Context, pagination *models.Pagination) ([]models.User, error) {
	return nil, nil
}
func (r *jwtUserRepo) ListWithFilters(ctx context.Context, pagination *models.Pagination, filters *types.UserSearchFilters) ([]models.User, error) {
	return nil, nil
}
func (r *jwtUserRepo) Create(ctx context.Context, user *models.User) error { return nil }
func (r *jwtUserRepo) CreateWithRoles(ctx context.Context, user *models.User, roleIDs []uint) error {
	return nil
}
func (r *jwtUserRepo) FindBasicByID(ctx context.Context, id uint) (*models.User, error) {
	return r.FindByID(ctx, id)
}
func (r *jwtUserRepo) Update(ctx context.Context, user *models.User) error { return nil }
func (r *jwtUserRepo) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	return nil
}
func (r *jwtUserRepo) UpdateStatus(ctx context.Context, id uint, status int) error { return nil }
func (r *jwtUserRepo) ExportWithFilters(ctx context.Context, filters *types.UserExportFilters) ([]models.User, error) {
	return nil, nil
}
func (r *jwtUserRepo) ReplaceUserRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	return nil
}
func (r *jwtUserRepo) Delete(ctx context.Context, id uint) error { return nil }
func (r *jwtUserRepo) UpdateLastLogin(ctx context.Context, userID uint) error {
	return nil
}

func TestJWT_acceptsValidBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &jwtUserRepo{user: &models.User{ID: 3, Username: "carol", Status: 1}}
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "0123456789abcdef0123456789abcdef", ExpireTime: 3600}}
	auth := services.NewAuthService(repo, nil, cfg)
	token, err := auth.RefreshToken(context.Background(), 3)
	require.NoError(t, err)

	r := gin.New()
	r.GET("/p", middleware.JWT(auth), func(c *gin.Context) {
		u, _ := c.Get("user")
		c.JSON(http.StatusOK, gin.H{"id": u.(*models.User).ID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":3`)
}

func TestJWT_rejectsMissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "0123456789abcdef0123456789abcdef", ExpireTime: int(time.Hour.Seconds())}}
	auth := services.NewAuthService(&jwtUserRepo{}, nil, cfg)
	r := gin.New()
	r.GET("/p", middleware.JWT(auth), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/p", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code) // business code in body
	assert.Contains(t, w.Body.String(), `"code":10005`)
}

func TestRBAC_deniesWithoutPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &denyStore{}
	auth := services.NewAuthService(&jwtUserRepo{user: &models.User{ID: 1}}, nil, &config.Config{
		JWT: config.JWTConfig{Secret: "0123456789abcdef0123456789abcdef", ExpireTime: 3600},
	})
	rbac := services.NewRBACService(store, auth, nil)

	r := gin.New()
	r.GET("/p", func(c *gin.Context) {
		c.Set("user", &models.User{ID: 1})
		c.Next()
	}, middleware.RBAC(rbac, "secret:op"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/p", nil)
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Body.String(), `"code":10006`)
}

type denyStore struct{}

func (d *denyStore) ListActivePermissions(ctx context.Context) ([]string, error) { return nil, nil }
func (d *denyStore) UserHasAdminRole(ctx context.Context, userID uint) (bool, error) {
	return false, nil
}
func (d *denyStore) ListUserPermissions(ctx context.Context, userID uint) ([]string, error) {
	return []string{"other"}, nil
}
func (d *denyStore) ListUserRolesWithMenus(ctx context.Context, userID uint) ([]models.Role, error) {
	return nil, nil
}
func (d *denyStore) ListVisibleMenus(ctx context.Context) ([]models.Menu, error) { return nil, nil }
