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
	token, ticket, proto := extractRealtimeToken(c, TokenExtractOptions{AllowQueryToken: true})
	require.Equal(t, "header-token", token)
	require.Empty(t, ticket)
	require.Empty(t, proto)
}

func TestExtractRealtimeToken_subprotocol(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/ws", nil)
	c.Request.Header.Set("Sec-WebSocket-Protocol", "access_token.abc.def.ghi")
	token, ticket, proto := extractRealtimeToken(c, TokenExtractOptions{})
	require.Equal(t, "abc.def.ghi", token)
	require.Empty(t, ticket)
	require.Equal(t, "access_token.abc.def.ghi", proto)
}

func TestExtractRealtimeToken_ticketQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/ws?ticket=tid-1", nil)
	token, ticket, proto := extractRealtimeToken(c, TokenExtractOptions{})
	require.Empty(t, token)
	require.Equal(t, "tid-1", ticket)
	require.Empty(t, proto)
}

func TestExtractRealtimeToken_queryBlockedWhenDisallowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/ws?token=legacy", nil)
	token, ticket, _ := extractRealtimeToken(c, TokenExtractOptions{AllowQueryToken: false})
	require.Empty(t, token)
	require.Empty(t, ticket)
}

func TestExtractRealtimeToken_queryFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/ws?token=legacy", nil)
	token, ticket, proto := extractRealtimeToken(c, TokenExtractOptions{AllowQueryToken: true})
	require.Equal(t, "legacy", token)
	require.Empty(t, ticket)
	require.Empty(t, proto)
}
