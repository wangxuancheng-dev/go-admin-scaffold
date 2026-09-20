package v1

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthHandler probes process and dependency health using injected clients.
type HealthHandler struct {
	DB    *gorm.DB
	Redis *redis.Client
}

func NewHealthHandler(db *gorm.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{DB: db, Redis: rdb}
}

func (h *HealthHandler) pingDB(ctx context.Context) bool {
	if h.DB == nil {
		return false
	}
	return h.DB.WithContext(ctx).Raw("SELECT 1").Error == nil
}

func (h *HealthHandler) pingRedis(ctx context.Context) bool {
	if h.Redis == nil {
		return false
	}
	return h.Redis.Ping(ctx).Err() == nil
}

// Liveness reports that the process is running (no dependency checks).
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

// Readiness reports whether the app can serve traffic (DB + Redis).
func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx := c.Request.Context()
	dbOK := h.pingDB(ctx)
	redisOK := h.pingRedis(ctx)
	body := gin.H{
		"status": "ready",
		"checks": gin.H{"database": dbOK, "redis": redisOK},
	}
	if !dbOK || !redisOK {
		body["status"] = "not_ready"
		c.JSON(http.StatusServiceUnavailable, body)
		return
	}
	c.JSON(http.StatusOK, body)
}

// HealthCheck keeps the legacy URL; semantics match Readiness.
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	h.Readiness(c)
}
