package middleware

import (
	"fmt"
	"runtime/debug"

	"go-admin-scaffold/pkg/logger"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

// Recovery returns a middleware that recovers from panics and writes a 500 response.
func Recovery(log logger.ContextLogger) gin.HandlerFunc {
	if log == nil {
		log = logger.Default()
	}
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := string(debug.Stack())
				log.Error(c.Request.Context(), "Panic recovered",
					"error", err,
					"stack", stack,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
				)

				var message string
				switch v := err.(type) {
				case error:
					message = v.Error()
				case string:
					message = v
				default:
					message = fmt.Sprintf("%v", v)
				}

				if !c.IsAborted() {
					response.Error(c, response.CodeServerError, message)
				}
			}
		}()

		c.Next()
	}
}
