package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-admin-scaffold/internal/api/admin/middleware"
	"go-admin-scaffold/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRecovery_catchesPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.Recovery(logger.Nop()))
	r.GET("/boom", func(c *gin.Context) {
		panic("explode")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/boom", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code) // envelope still 200 with business code
	assert.Contains(t, w.Body.String(), "explode")
	assert.Contains(t, w.Body.String(), `"code":10002`)
}

func TestJWT_nilAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/p", middleware.JWT(nil, logger.Nop()), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/p", nil)
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Body.String(), `"code":10002`)
}
