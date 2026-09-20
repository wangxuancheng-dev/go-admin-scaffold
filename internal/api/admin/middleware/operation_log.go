package middleware

import (
	"bytes"
	"io"
	"time"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/services"

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

// OperationLog records admin API operation logs using the process-scoped LogService.
func OperationLog(logSvc *services.LogService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if shouldSkipLogging(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()

		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		blw := &bodyLogWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = blw

		var userID uint
		var username string
		if user, exists := c.Get("user"); exists {
			if userModel, ok := user.(*models.User); ok {
				userID = userModel.ID
				username = userModel.Username
			}
		}

		c.Next()

		duration := time.Since(start).Milliseconds()
		log := &models.OperationLog{
			UserID:        userID,
			Username:      username,
			IP:            c.ClientIP(),
			Method:        c.Request.Method,
			Path:          c.Request.URL.Path,
			Action:        getActionFromPath(c.Request.URL.Path),
			Module:        getModuleFromPath(c.Request.URL.Path),
			RequestParams: string(requestBody),
			Status:        1,
			OperationTime: models.CustomTime(time.Now()),
			Duration:      duration,
			UserAgent:     c.Request.UserAgent(),
		}

		if len(c.Errors) > 0 {
			log.Status = 0
			log.ErrorMessage = c.Errors.String()
		}

		if logSvc != nil && userID > 0 {
			logSvc.RecordOperationLog(c.Request.Context(), log)
		}
	}
}

func shouldSkipLogging(path string) bool {
	skipPaths := []string{
		"/api/admin/v1/logs",
		"/api/open/v1/public/live",
		"/api/open/v1/public/ready",
		"/api/admin/v1/auth/refresh",
	}
	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

func getModuleFromPath(path string) string {
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

func getActionFromPath(path string) string {
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

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
