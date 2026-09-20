package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExtractRealtimeToken_prefersBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/ws?token=query-token", nil)
	c.Request.Header.Set("Authorization", "Bearer header-token")
	token, proto := extractRealtimeToken(c)
	require.Equal(t, "header-token", token)
	require.Empty(t, proto)
}

func TestExtractRealtimeToken_subprotocol(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/ws", nil)
	c.Request.Header.Set("Sec-WebSocket-Protocol", "access_token.abc.def.ghi")
	token, proto := extractRealtimeToken(c)
	require.Equal(t, "abc.def.ghi", token)
	require.Equal(t, "access_token.abc.def.ghi", proto)
}

func TestExtractRealtimeToken_queryFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/ws?token=legacy", nil)
	token, proto := extractRealtimeToken(c)
	require.Equal(t, "legacy", token)
	require.Empty(t, proto)
}
