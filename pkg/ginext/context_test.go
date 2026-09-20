package ginext_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-admin-scaffold/pkg/ginext"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetService_ok(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		c.Set("svc", "hello")
		v, ok := ginext.GetService[string](c, "svc")
		assert.True(t, ok)
		assert.Equal(t, "hello", v)
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetService_missing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		_, ok := ginext.GetService[string](c, "missing")
		assert.False(t, ok)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Body.String(), `"code":10002`)
}

func TestGetService_typeMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		c.Set("svc", 123)
		_, ok := ginext.GetService[string](c, "svc")
		assert.False(t, ok)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Body.String(), `"code":10002`)
}
