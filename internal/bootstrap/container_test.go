package bootstrap_test

import (
	"context"
	"testing"
	"time"

	"go-admin-scaffold/internal/bootstrap"
	"go-admin-scaffold/internal/config"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNewContainer_wiresServices(t *testing.T) {
	// Empty *gorm.DB is enough: constructors only store the handle.
	db := &gorm.DB{}
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:     "0123456789abcdef0123456789abcdef",
			ExpireTime: 3600,
		},
		Storage: config.StorageConfig{
			Driver: "local",
			Local:  config.LocalConfig{Path: t.TempDir()},
		},
	}

	c, err := bootstrap.NewContainer(cfg, db, nil)
	require.NoError(t, err)
	require.NotNil(t, c.Auth)
	require.NotNil(t, c.User)
	require.NotNil(t, c.RBAC)
	require.NotNil(t, c.Role)
	require.NotNil(t, c.Menu)
	require.NotNil(t, c.Todo)
	require.NotNil(t, c.Log)
	require.NotNil(t, c.Realtime)
	require.NotNil(t, c.Storage)
	require.Equal(t, db, c.DB)
	require.Nil(t, c.Redis)
}

func TestNewContainerWithCache_setsCache(t *testing.T) {
	db := &gorm.DB{}
	cfg := &config.Config{
		JWT:     config.JWTConfig{Secret: "0123456789abcdef0123456789abcdef", ExpireTime: 3600},
		Storage: config.StorageConfig{Driver: "local", Local: config.LocalConfig{Path: t.TempDir()}},
	}
	cch := &stubCache{}
	c, err := bootstrap.NewContainerWithCache(cfg, db, nil, cch)
	require.NoError(t, err)
	require.Equal(t, cch, c.Cache)
}

type stubCache struct{}

func (s *stubCache) Get(ctx context.Context, key string) (string, error) { return "", nil }
func (s *stubCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return nil
}
func (s *stubCache) Delete(ctx context.Context, key string) error { return nil }
func (s *stubCache) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func TestNewContainer_requiresDeps(t *testing.T) {
	_, err := bootstrap.NewContainer(nil, nil, nil)
	require.Error(t, err)
	_, err = bootstrap.NewContainer(&config.Config{}, nil, nil)
	require.Error(t, err)
}
