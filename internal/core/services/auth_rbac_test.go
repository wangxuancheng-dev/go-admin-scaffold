package services

import (
	"context"
	"testing"
	"time"

	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/types"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type stubUserRepo struct {
	user *models.User
	err  error
}

func (s *stubUserRepo) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.user != nil && s.user.Username == username {
		return s.user, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *stubUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return nil, gorm.ErrRecordNotFound
}

func (s *stubUserRepo) FindByID(ctx context.Context, id uint) (*models.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.user != nil && s.user.ID == id {
		return s.user, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *stubUserRepo) ListWithRoles(ctx context.Context, pagination *models.Pagination) ([]models.User, error) {
	return nil, nil
}

func (s *stubUserRepo) ListWithFilters(ctx context.Context, pagination *models.Pagination, filters *types.UserSearchFilters) ([]models.User, error) {
	return nil, nil
}

func (s *stubUserRepo) Create(ctx context.Context, user *models.User) error { return nil }
func (s *stubUserRepo) Update(ctx context.Context, user *models.User) error { return nil }
func (s *stubUserRepo) Delete(ctx context.Context, id uint) error           { return nil }
func (s *stubUserRepo) UpdateLastLogin(ctx context.Context, userID uint) error {
	return nil
}
func (s *stubUserRepo) GetDB() *gorm.DB { return nil }

func testAuthConfig(secret string, superIDs []uint) *config.Config {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:     secret,
			ExpireTime: 3600,
			Issuer:     "test",
		},
	}
	cfg.SuperAdminIDs = superIDs
	return cfg
}

func TestAuthService_IsSuperAdmin(t *testing.T) {
	svc := NewAuthService(&stubUserRepo{}, nil, testAuthConfig("0123456789abcdef0123456789abcdef", []uint{1, 9}))
	assert.True(t, svc.IsSuperAdmin(1))
	assert.True(t, svc.IsSuperAdmin(9))
	assert.False(t, svc.IsSuperAdmin(2))
}

func TestAuthService_ValidateToken_roundTrip(t *testing.T) {
	repo := &stubUserRepo{user: &models.User{ID: 7, Username: "alice", Status: 1}}
	svc := NewAuthService(repo, nil, testAuthConfig("0123456789abcdef0123456789abcdef", nil))

	token, err := svc.generateToken(repo.user)
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "alice", claims["username"])

	user, err := svc.GetUserFromClaims(context.Background(), claims)
	require.NoError(t, err)
	assert.Equal(t, uint(7), user.ID)
}

func TestAuthService_ValidateToken_rejectsWrongAlg(t *testing.T) {
	svc := NewAuthService(&stubUserRepo{}, nil, testAuthConfig("0123456789abcdef0123456789abcdef", nil))
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"user_id":  1,
		"username": "x",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = svc.ValidateToken(signed)
	assert.Error(t, err)
}

func TestAuthService_Login_inactive(t *testing.T) {
	hasher := NewAuthService(&stubUserRepo{}, nil, testAuthConfig("0123456789abcdef0123456789abcdef", nil))
	hash, err := hasher.HashPassword("secret")
	require.NoError(t, err)
	repo := &stubUserRepo{user: &models.User{ID: 1, Username: "bob", Password: hash, Status: 0}}
	svc := NewAuthService(repo, nil, testAuthConfig("0123456789abcdef0123456789abcdef", nil))

	_, err = svc.Login(context.Background(), &LoginRequest{
		Username: "bob", Password: "secret", CaptchaID: "x", CaptchaCode: "y",
	})
	assert.ErrorIs(t, err, ErrUserInactive)
}

func TestRBACService_CheckPermission_superAdmin(t *testing.T) {
	auth := NewAuthService(&stubUserRepo{}, nil, testAuthConfig("0123456789abcdef0123456789abcdef", []uint{42}))
	rbac := NewRBACService(nil, auth, nil)

	ok, err := rbac.CheckPermission(context.Background(), &models.User{ID: 42}, "anything")
	require.NoError(t, err)
	assert.True(t, ok)
}
