package middleware

import (
	"bytes"
	"io"
	"time"

	"app/internal/core/models"
	"app/internal/core/services"

	"github.com/gin-gonic/gin"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// OperationLog returns a middleware that records operation logs
func OperationLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for certain paths
		if shouldSkipLogging(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()

		// Read request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Create a custom response writer to capture the response
		blw := &bodyLogWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = blw

		// Get user information from context
		var userID uint
		var username string

		if user, exists := c.Get("user"); exists {
			if userModel, ok := user.(*models.User); ok {
				userID = userModel.ID
				username = userModel.Username
			}
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start).Milliseconds()

		// Create operation log
		log := &models.OperationLog{
			UserID:        userID,
			Username:      username,
			IP:            c.ClientIP(),
			Method:        c.Request.Method,
			Path:          c.Request.URL.Path,
			Action:        getActionFromPath(c.Request.URL.Path),
			Module:        getModuleFromPath(c.Request.URL.Path),
			RequestParams: string(requestBody),
			Status:        1, // Default to success
			OperationTime: models.CustomTime(time.Now()),
			Duration:      duration,
			UserAgent:     c.Request.UserAgent(),
		}

		// Update status and error message if there was an error
		if len(c.Errors) > 0 {
			log.Status = 0
			log.ErrorMessage = c.Errors.String()
		}

		// Get log service and record the operation
		if logSvc, exists := c.Get("logService"); exists {
			if ls, ok := logSvc.(*services.LogService); ok && userID > 0 {
				ls.RecordOperationLog(c.Request.Context(), log)
			}
		}
	}
}

// shouldSkipLogging returns true if the path should not be logged
func shouldSkipLogging(path string) bool {
	// Skip logging for these paths
	skipPaths := []string{
		"/api/admin/v1/logs",         // Skip logging the log endpoints themselves
		"/api/open/v1/public/health", // Skip health / readiness probes
		"/api/open/v1/public/live",
		"/api/open/v1/public/ready",
		"/api/admin/v1/auth/refresh", // Skip token refresh endpoint
	}

	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// getModuleFromPath extracts the module name from the URL path
func getModuleFromPath(path string) string {
	// Example: /api/admin/v1/users -> users
	// You can implement more sophisticated logic here
	switch {
	case contains(path, "/users"):
		return "users"
	case contains(path, "/roles"):
		return "roles"
	case contains(path, "/auth"):
		return "auth"
	default:
		return "other"
	}
}

// getActionFromPath extracts the action from the URL path and HTTP method
func getActionFromPath(path string) string {
	// Example: GET /users -> list users
	// You can implement more sophisticated logic here
	switch {
	case contains(path, "/users"):
		return "user management"
	case contains(path, "/roles"):
		return "role management"
	case contains(path, "/auth/login"):
		return "user login"
	default:
		return "other"
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
