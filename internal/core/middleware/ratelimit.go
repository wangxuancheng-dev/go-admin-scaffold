package middleware

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// RateLimiter is an in-process token-bucket limiter (fallback when Redis is unavailable).
type RateLimiter struct {
	limiters map[string]map[string]*ipLimiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	ttl      time.Duration
	cleanup  time.Duration
}

type ipLimiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// NewRateLimiter creates an in-memory rate limiter with periodic cleanup.
func NewRateLimiter(r rate.Limit, b int, ttl time.Duration) *RateLimiter {
	limiter := &RateLimiter{
		limiters: make(map[string]map[string]*ipLimiter),
		rate:     r,
		burst:    b,
		ttl:      ttl,
		cleanup:  ttl * 2,
	}
	go limiter.cleanupLoop()
	return limiter
}

func (l *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for path, ipLimiters := range l.limiters {
			for ip, lim := range ipLimiters {
				if now.Sub(lim.lastAccess) > l.ttl {
					delete(ipLimiters, ip)
				}
			}
			if len(ipLimiters) == 0 {
				delete(l.limiters, path)
			}
		}
		l.mu.Unlock()
	}
}

func (l *RateLimiter) allow(path, ip string) bool {
	return l.getLimiter(path, ip).Allow()
}

func (l *RateLimiter) getLimiter(path, ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	ipLimiters, exists := l.limiters[path]
	if !exists {
		ipLimiters = make(map[string]*ipLimiter)
		l.limiters[path] = ipLimiters
	}

	lim, exists := ipLimiters[ip]
	if !exists {
		lim = &ipLimiter{
			limiter:    rate.NewLimiter(l.rate, l.burst),
			lastAccess: time.Now(),
		}
		ipLimiters[ip] = lim
	} else {
		lim.lastAccess = time.Now()
	}
	return lim.limiter
}

// RateLimit is an in-memory limiter (single instance only). Prefer RateLimitRedis in production.
func RateLimit(r rate.Limit, b int) gin.HandlerFunc {
	limiter := NewRateLimiter(r, b, time.Hour)
	return func(c *gin.Context) {
		path := c.FullPath()
		ip := c.ClientIP()
		if !limiter.allow(path, ip) {
			response.TooManyRequests(c, "Too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}

func memoryFallback(window time.Duration, max int) *RateLimiter {
	per := window / time.Duration(max)
	if per <= 0 {
		per = time.Millisecond
	}
	return NewRateLimiter(rate.Every(per), max, time.Hour)
}

// RateLimitRedis applies a fixed-window counter shared across instances via Redis.
// On Redis errors it falls back to a local hard limit (never fail-open).
func RateLimitRedis(rdb *redis.Client, window time.Duration, max int) gin.HandlerFunc {
	if max < 1 {
		max = 1
	}
	if window <= 0 {
		window = time.Second
	}
	local := memoryFallback(window, max)
	if rdb == nil {
		return func(c *gin.Context) {
			path := c.FullPath()
			if path == "" {
				path = c.Request.URL.Path
			}
			if !local.allow(path, c.ClientIP()) {
				response.TooManyRequests(c, "Too many requests")
				c.Abort()
				return
			}
			c.Next()
		}
	}

	prefix := "ratelimit:"
	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		ip := c.ClientIP()
		key := prefix + path + ":" + ip
		ctx := c.Request.Context()

		n, err := incrWithExpire(ctx, rdb, key, window)
		if err != nil {
			// Fail closed via local limiter — never silently allow bursts.
			if !local.allow(path, ip) {
				response.TooManyRequests(c, "Too many requests")
				c.Abort()
				return
			}
			c.Next()
			return
		}
		if n > int64(max) {
			response.TooManyRequests(c, "Too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}

func incrWithExpire(ctx context.Context, rdb *redis.Client, key string, window time.Duration) (int64, error) {
	pipe := rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Result()
}

// FormatRateLimitKey is exported for tests.
func FormatRateLimitKey(path, ip string) string {
	return fmt.Sprintf("ratelimit:%s:%s", path, ip)
}

// ParsePositiveInt is a tiny helper kept for callers that need string burst config.
func ParsePositiveInt(s string, fallback int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}
