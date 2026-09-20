package middleware

import (
	"fmt"
	"runtime/debug"

	"go-admin-scaffold/pkg/logger"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

// Recovery returns a middleware that recovers from any panics and writes a 500 if there was one.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the error and stack trace
				stack := string(debug.Stack())
				logger.Error(c.Request.Context(), "Panic recovered",
					"error", err,
					"stack", stack,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
				)

				// Get error message
				var message string
				switch v := err.(type) {
				case error:
					message = v.Error()
				case string:
					message = v
				default:
					message = fmt.Sprintf("%v", v)
				}

				// Respond with 500 error
				if !c.IsAborted() {
					response.Error(c, response.CodeServerError, message)
				}
			}
		}()

		c.Next()
	}
}
