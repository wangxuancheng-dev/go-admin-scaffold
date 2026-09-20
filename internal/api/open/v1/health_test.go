package v1_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	openv1 "go-admin-scaffold/internal/api/open/v1"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler_livenessAlwaysOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := openv1.NewHealthHandler(nil, nil)
	r := gin.New()
	r.GET("/live", h.Liveness)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/live", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthHandler_readinessFailsWithoutDeps(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := openv1.NewHealthHandler(nil, nil)
	r := gin.New()
	r.GET("/ready", h.Readiness)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ready", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "not_ready", body["status"])
}

func TestOAuthStubs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/oauth", openv1.GithubOAuth)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/oauth", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
