package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	httpRequests atomic.Uint64
	httpInFlight atomic.Int64
	httpErrors   atomic.Uint64
	startedAt    = time.Now()

	latencyMu     sync.Mutex
	latencyCount  uint64
	latencySumMs  float64
	latencyBuckets = [...]float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000}
	latencyObs    [len(latencyBuckets) + 1]uint64

	statusMu sync.Mutex
	byStatus = map[string]uint64{}
	byMethod = map[string]uint64{}
)

// Middleware records lightweight request counters and latency for /metrics.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}
		start := time.Now()
		httpInFlight.Add(1)
		httpRequests.Add(1)
		c.Next()
		httpInFlight.Add(-1)
		status := c.Writer.Status()
		if status >= 500 {
			httpErrors.Add(1)
		}
		observe(c.Request.Method, status, time.Since(start))
	}
}

func observe(method string, status int, d time.Duration) {
	ms := float64(d.Milliseconds())
	class := statusClass(status)
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "UNKNOWN"
	}

	statusMu.Lock()
	byStatus[class]++
	byMethod[method]++
	statusMu.Unlock()

	latencyMu.Lock()
	latencyCount++
	latencySumMs += ms
	placed := false
	for i, bound := range latencyBuckets {
		if ms <= bound {
			latencyObs[i]++
			placed = true
			break
		}
	}
	if !placed {
		latencyObs[len(latencyBuckets)]++
	}
	latencyMu.Unlock()
}

func statusClass(status int) string {
	switch {
	case status >= 500:
		return "5xx"
	case status >= 400:
		return "4xx"
	case status >= 300:
		return "3xx"
	case status >= 200:
		return "2xx"
	default:
		return "1xx"
	}
}

// Auth protects /metrics when token is non-empty.
func Auth(token string) gin.HandlerFunc {
	token = strings.TrimSpace(token)
	return func(c *gin.Context) {
		if token == "" {
			c.Next()
			return
		}
		got := strings.TrimSpace(c.GetHeader("X-Metrics-Token"))
		if got == "" {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				got = strings.TrimSpace(auth[7:])
			}
		}
		if got == "" || got != token {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

// Handler exposes Prometheus text exposition format (no external deps).
func Handler(c *gin.Context) {
	uptime := time.Since(startedAt).Seconds()

	statusMu.Lock()
	statusCopy := make(map[string]uint64, len(byStatus))
	for k, v := range byStatus {
		statusCopy[k] = v
	}
	methodCopy := make(map[string]uint64, len(byMethod))
	for k, v := range byMethod {
		methodCopy[k] = v
	}
	statusMu.Unlock()

	latencyMu.Lock()
	count := latencyCount
	sum := latencySumMs
	obs := latencyObs
	latencyMu.Unlock()

	var b strings.Builder
	b.WriteString("# HELP go_admin_http_requests_total Total HTTP requests handled.\n")
	b.WriteString("# TYPE go_admin_http_requests_total counter\n")
	b.WriteString("go_admin_http_requests_total " + strconv.FormatUint(httpRequests.Load(), 10) + "\n")
	b.WriteString("# HELP go_admin_http_inflight Current in-flight HTTP requests.\n")
	b.WriteString("# TYPE go_admin_http_inflight gauge\n")
	b.WriteString("go_admin_http_inflight " + strconv.FormatInt(httpInFlight.Load(), 10) + "\n")
	b.WriteString("# HELP go_admin_http_errors_total HTTP responses with status >= 500.\n")
	b.WriteString("# TYPE go_admin_http_errors_total counter\n")
	b.WriteString("go_admin_http_errors_total " + strconv.FormatUint(httpErrors.Load(), 10) + "\n")
	b.WriteString("# HELP go_admin_uptime_seconds Process uptime in seconds.\n")
	b.WriteString("# TYPE go_admin_uptime_seconds gauge\n")
	b.WriteString("go_admin_uptime_seconds " + strconv.FormatFloat(uptime, 'f', 3, 64) + "\n")

	b.WriteString("# HELP go_admin_http_requests_by_status_total Requests by HTTP status class.\n")
	b.WriteString("# TYPE go_admin_http_requests_by_status_total counter\n")
	for _, class := range []string{"1xx", "2xx", "3xx", "4xx", "5xx"} {
		if v, ok := statusCopy[class]; ok {
			b.WriteString(`go_admin_http_requests_by_status_total{class="` + class + `"} ` + strconv.FormatUint(v, 10) + "\n")
		}
	}

	b.WriteString("# HELP go_admin_http_requests_by_method_total Requests by HTTP method.\n")
	b.WriteString("# TYPE go_admin_http_requests_by_method_total counter\n")
	for method, v := range methodCopy {
		b.WriteString(`go_admin_http_requests_by_method_total{method="` + method + `"} ` + strconv.FormatUint(v, 10) + "\n")
	}

	b.WriteString("# HELP go_admin_http_request_duration_ms Request latency in milliseconds.\n")
	b.WriteString("# TYPE go_admin_http_request_duration_ms histogram\n")
	var cumulative uint64
	for i, bound := range latencyBuckets {
		cumulative += obs[i]
		b.WriteString(`go_admin_http_request_duration_ms_bucket{le="` + strconv.FormatFloat(bound, 'f', -1, 64) + `"} ` + strconv.FormatUint(cumulative, 10) + "\n")
	}
	cumulative += obs[len(latencyBuckets)]
	b.WriteString(`go_admin_http_request_duration_ms_bucket{le="+Inf"} ` + strconv.FormatUint(cumulative, 10) + "\n")
	b.WriteString("go_admin_http_request_duration_ms_sum " + strconv.FormatFloat(sum, 'f', 3, 64) + "\n")
	b.WriteString("go_admin_http_request_duration_ms_count " + strconv.FormatUint(count, 10) + "\n")

	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(b.String()))
}
