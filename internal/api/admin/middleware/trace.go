package middleware

import (
	"go-admin-scaffold/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	TraceIDKey    = "X-Trace-ID"
	TraceIDCtxKey = "trace_id"
)

// Trace adds a trace ID to each request
func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get trace ID from header or generate new one
		traceID := c.GetHeader(TraceIDKey)
		if traceID == "" {
			traceID = generateTraceID()
		}

		// Set trace ID in context and header
		c.Set(TraceIDCtxKey, traceID)
		c.Header(TraceIDKey, traceID)

		// Add trace ID to logger context
		ctx := logger.WithField(c.Request.Context(), "trace_id", traceID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// GetTraceID gets trace ID from gin context
func GetTraceID(c *gin.Context) string {
	if traceID, exists := c.Get(TraceIDCtxKey); exists {
		return traceID.(string)
	}
	return ""
}

// generateTraceID generates a new trace ID
func generateTraceID() string {
	return uuid.New().String()
}
