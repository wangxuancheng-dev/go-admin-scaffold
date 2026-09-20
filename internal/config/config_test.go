package config_test

import (
	"testing"

	"go-admin-scaffold/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate_requiresJWTAndAddress(t *testing.T) {
	cfg := &config.Config{}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jwt.secret")

	cfg.JWT.Secret = "short"
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server.address")

	cfg.Server.Address = ":8080"
	require.NoError(t, cfg.Validate())
}

func TestConfigValidate_productionHardening(t *testing.T) {
	cfg := &config.Config{
		App:    config.AppConfig{Env: "production"},
		JWT:    config.JWTConfig{Secret: "0123456789abcdef0123456789abcdef"},
		Server: config.ServerConfig{Address: ":8080"},
		CORS:   config.CORSConfig{AllowOrigins: []string{"*"}},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database.host")

	cfg.Database.Host = "localhost"
	cfg.Database.Database = "go_admin"
	cfg.Redis.Host = "localhost"
	cfg.CORS.AllowOrigins = []string{"https://admin.example.com"}
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metrics.token")

	cfg.Metrics.Token = "scrape-secret"
	require.NoError(t, cfg.Validate())
}

func TestConfigValidate_rejectsCredentialsWithWildcard(t *testing.T) {
	cfg := &config.Config{
		JWT:    config.JWTConfig{Secret: "dev-secret"},
		Server: config.ServerConfig{Address: ":8080"},
		CORS: config.CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowCredentials: true,
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allow_credentials")
}

func TestConfig_SchedulerRunInServerDefault(t *testing.T) {
	cfg := &config.Config{}
	require.True(t, cfg.SchedulerRunInServer())

	off := false
	cfg.Scheduler.RunInServer = &off
	require.False(t, cfg.SchedulerRunInServer())
}
