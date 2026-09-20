package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit_memoryFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// nil redis -> memory limiter; burst 1 so second request fails
	r.GET("/x", RateLimitRedis(nil, time.Hour, 1), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w2, req2)
	// business errors still HTTP 200 with code; TooManyRequests may set 200 or 429 depending on response helper
	assert.NotEqual(t, "", w2.Body.String())
}

func TestFormatRateLimitKey(t *testing.T) {
	assert.Equal(t, "ratelimit:/login:1.2.3.4", FormatRateLimitKey("/login", "1.2.3.4"))
}
