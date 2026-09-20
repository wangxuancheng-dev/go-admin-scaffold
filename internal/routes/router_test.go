package routes_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-admin-scaffold/internal/bootstrap"
	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/routes"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSetupRoutes_livenessRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	cfg := &config.Config{
		App:    config.AppConfig{Env: "test"},
		JWT:    config.JWTConfig{Secret: "test-secret-at-least-32-chars!!"},
		Server: config.ServerConfig{Address: ":0"},
		Storage: config.StorageConfig{
			Driver: "local",
			Local:  config.LocalConfig{Path: t.TempDir()},
		},
	}
	enabled := false
	cfg.Metrics.Enabled = &enabled

	c, err := bootstrap.NewContainer(cfg, db, nil)
	require.NoError(t, err)

	r := gin.New()
	require.NoError(t, routes.SetupRoutes(r, c))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/open/v1/public/live", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "alive")
}
