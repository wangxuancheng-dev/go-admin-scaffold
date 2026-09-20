package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccessAndPageSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ok", func(c *gin.Context) {
		response.Success(c, gin.H{"a": 1})
	})
	r.GET("/page", func(c *gin.Context) {
		response.PageSuccess(c, []string{"x"}, 1, 1, 10)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ok", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var env response.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 0, env.Code)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/page", nil)
	r.ServeHTTP(w2, req2)
	var page map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &page))
	assert.Equal(t, float64(0), page["code"])
}

func TestErrorHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		path string
		fn   gin.HandlerFunc
		code int
	}{
		{"/u", func(c *gin.Context) { response.UnauthorizedError(c) }, response.CodeUnauthorized},
		{"/f", func(c *gin.Context) { response.ForbiddenError(c) }, response.CodeForbidden},
		{"/n", func(c *gin.Context) { response.NotFoundError(c) }, response.CodeNotFound},
		{"/t", func(c *gin.Context) { response.TooManyRequests(c, "") }, response.CodeTooManyRequests},
	}
	r := gin.New()
	for _, tc := range cases {
		r.GET(tc.path, tc.fn)
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, tc.path, nil)
		r.ServeHTTP(w, req)
		var env response.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
		assert.Equal(t, tc.code, env.Code, tc.path)
	}
}
