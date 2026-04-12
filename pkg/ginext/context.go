package ginext

import (
	"app/pkg/response"

	"github.com/gin-gonic/gin"
)

// GetService returns a typed value from gin context (e.g. injected services).
// On missing key or type mismatch it responds with 500, aborts the chain, and returns ok=false.
func GetService[T any](c *gin.Context, key string) (v T, ok bool) {
	var zero T
	raw, exists := c.Get(key)
	if !exists {
		response.ServerError(c)
		c.Abort()
		return zero, false
	}
	v, typeOK := raw.(T)
	if !typeOK {
		response.ServerError(c)
		c.Abort()
		return zero, false
	}
	return v, true
}
