package middleware

import (
	"app/pkg/response"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ipLimiter 包含限流器和最后访问时间
type ipLimiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// RateLimiter 限流器
type RateLimiter struct {
	limiters map[string]map[string]*ipLimiter // path -> ip -> limiter
	mu       *sync.RWMutex
	rate     rate.Limit
	burst    int
	ttl      time.Duration
	cleanup  time.Duration
}

// NewRateLimiter 创建限流器
func NewRateLimiter(r rate.Limit, b int, ttl time.Duration) *RateLimiter {
	limiter := &RateLimiter{
		limiters: make(map[string]map[string]*ipLimiter),
		mu:       &sync.RWMutex{},
		rate:     r,
		burst:    b,
		ttl:      ttl,
		cleanup:  ttl * 2,
	}

	// 启动清理过期限流器的goroutine
	go limiter.cleanupLoop()

	return limiter
}

// cleanupLoop 定期清理过期的限流器
func (l *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for path, ipLimiters := range l.limiters {
			for ip, limiter := range ipLimiters {
				if now.Sub(limiter.lastAccess) > l.ttl {
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

// getLimiter 获取指定路径和IP对应的限流器
func (l *RateLimiter) getLimiter(path, ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 获取或创建路径的IP限流器映射
	ipLimiters, exists := l.limiters[path]
	if !exists {
		ipLimiters = make(map[string]*ipLimiter)
		l.limiters[path] = ipLimiters
	}

	// 获取或创建IP的限流器
	limiter, exists := ipLimiters[ip]
	if !exists {
		limiter = &ipLimiter{
			limiter:    rate.NewLimiter(l.rate, l.burst),
			lastAccess: time.Now(),
		}
		ipLimiters[ip] = limiter
	} else {
		limiter.lastAccess = time.Now()
	}

	return limiter.limiter
}

// RateLimit 限流中间件
func RateLimit(r rate.Limit, b int) gin.HandlerFunc {
	limiter := NewRateLimiter(r, b, time.Hour)

	return func(c *gin.Context) {
		path := c.FullPath()
		ip := c.ClientIP()

		if !limiter.getLimiter(path, ip).Allow() {
			response.TooManyRequests(c, "Too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}
