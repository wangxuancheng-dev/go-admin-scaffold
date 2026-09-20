package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-admin-scaffold/internal/core/metrics"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsHandler_exposesCounters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(metrics.Middleware())
	r.GET("/metrics", metrics.Handler)
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	mw := httptest.NewRecorder()
	mreq, _ := http.NewRequest(http.MethodGet, "/metrics", nil)
	r.ServeHTTP(mw, mreq)
	assert.Equal(t, http.StatusOK, mw.Code)
	assert.Contains(t, mw.Body.String(), "go_admin_http_requests_total")
	assert.Contains(t, mw.Body.String(), "go_admin_uptime_seconds")
	assert.Contains(t, mw.Body.String(), "go_admin_http_request_duration_ms_bucket")
	assert.Contains(t, mw.Body.String(), `go_admin_http_requests_by_status_total{class="2xx"}`)
}

func TestMetricsAuth_requiresToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/metrics", metrics.Auth("secret"), metrics.Handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/metrics", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("X-Metrics-Token", "secret")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer secret")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
