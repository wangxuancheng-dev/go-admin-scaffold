package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-admin-scaffold/internal/core/metrics"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
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
}
