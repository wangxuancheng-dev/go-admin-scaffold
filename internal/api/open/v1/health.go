package v1

import (
	"context"

	"app/pkg/database"
	"app/pkg/redis"

	"github.com/gin-gonic/gin"
)

func pingDB(ctx context.Context) bool {
	db := database.GetDB()
	if db == nil {
		return false
	}
	return db.WithContext(ctx).Raw("SELECT 1").Error == nil
}

func pingRedis(ctx context.Context) bool {
	return redis.GetClient().Ping(ctx).Err() == nil
}

// Liveness reports that the process is running (no dependency checks).
func Liveness(c *gin.Context) {
	c.JSON(200, gin.H{"status": "alive"})
}

// Readiness reports whether the app can serve traffic (DB + Redis). Returns 503 if any check fails.
func Readiness(c *gin.Context) {
	ctx := c.Request.Context()
	dbOK := pingDB(ctx)
	redisOK := pingRedis(ctx)
	if !dbOK || !redisOK {
		c.JSON(503, gin.H{
			"status": "not_ready",
			"checks": gin.H{
				"database": dbOK,
				"redis":    redisOK,
			},
		})
		return
	}
	c.JSON(200, gin.H{
		"status": "ready",
		"checks": gin.H{
			"database": dbOK,
			"redis":    redisOK,
		},
	})
}

// HealthCheck is kept for backward-compatible URLs; semantics match Readiness (503 when dependencies are down).
func HealthCheck(c *gin.Context) {
	Readiness(c)
}
