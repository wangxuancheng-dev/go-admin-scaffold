package metrics

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	httpRequests   atomic.Uint64
	httpInFlight   atomic.Int64
	httpErrors     atomic.Uint64
	startedAt      = time.Now()
)

// Middleware records lightweight request counters for /metrics.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}
		httpInFlight.Add(1)
		httpRequests.Add(1)
		c.Next()
		httpInFlight.Add(-1)
		if c.Writer.Status() >= 500 {
			httpErrors.Add(1)
		}
	}
}

// Handler exposes Prometheus text exposition format (no external deps).
func Handler(c *gin.Context) {
	uptime := time.Since(startedAt).Seconds()
	body := "" +
		"# HELP go_admin_http_requests_total Total HTTP requests handled.\n" +
		"# TYPE go_admin_http_requests_total counter\n" +
		"go_admin_http_requests_total " + strconv.FormatUint(httpRequests.Load(), 10) + "\n" +
		"# HELP go_admin_http_inflight Current in-flight HTTP requests.\n" +
		"# TYPE go_admin_http_inflight gauge\n" +
		"go_admin_http_inflight " + strconv.FormatInt(httpInFlight.Load(), 10) + "\n" +
		"# HELP go_admin_http_errors_total HTTP responses with status >= 500.\n" +
		"# TYPE go_admin_http_errors_total counter\n" +
		"go_admin_http_errors_total " + strconv.FormatUint(httpErrors.Load(), 10) + "\n" +
		"# HELP go_admin_uptime_seconds Process uptime in seconds.\n" +
		"# TYPE go_admin_uptime_seconds gauge\n" +
		"go_admin_uptime_seconds " + strconv.FormatFloat(uptime, 'f', 3, 64) + "\n"
	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(body))
}
