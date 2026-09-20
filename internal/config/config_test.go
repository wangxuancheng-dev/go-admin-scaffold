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
}
