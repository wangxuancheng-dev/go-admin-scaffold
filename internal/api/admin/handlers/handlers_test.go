package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-admin-scaffold/internal/api/admin/handlers"
	"go-admin-scaffold/internal/core/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func withUser(user *models.User) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user", user)
		c.Next()
	}
}

func TestWSHandler_requiresToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewWSHandler(nil, nil)
	r := gin.New()
	r.GET("/ws", h.HandleWebSocket)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ws", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "token is required")
}

func TestWSHandler_nilAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewWSHandler(nil, nil)
	r := gin.New()
	r.GET("/ws", h.HandleWebSocket)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ws?token=x", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"code"`)
}

func TestWSHandler_joinLeaveFromJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewWSHandler(nil, nil)
	user := &models.User{ID: 1, Username: "alice"}
	r := gin.New()
	r.POST("/join", withUser(user), h.JoinGroup)
	r.POST("/leave", withUser(user), h.LeaveGroup)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/join", nil)
	r.ServeHTTP(w, req)
	require.Contains(t, w.Body.String(), "group_id is required")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/join?group_id=g1", nil)
	r.ServeHTTP(w, req)
	require.Contains(t, w.Body.String(), "Successfully joined")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/leave?group_id=g1", nil)
	r.ServeHTTP(w, req)
	require.Contains(t, w.Body.String(), "Successfully left")
}

func TestWSHandler_joinUnauthorizedWithoutUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewWSHandler(nil, nil)
	r := gin.New()
	r.POST("/join", h.JoinGroup)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/join?group_id=g1", nil)
	r.ServeHTTP(w, req)
	require.Contains(t, w.Body.String(), `"code"`)
}

func TestWSHandler_sendMessageSetsFrom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewWSHandler(nil, nil)
	user := &models.User{ID: 2, Username: "bob"}
	r := gin.New()
	r.POST("/send", withUser(user), h.SendMessage)

	body, _ := json.Marshal(map[string]any{"type": 1, "content": "hi", "to": "g1"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Message sent")
}

func TestSSEHandler_requiresToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewSSEHandler(nil)
	r := gin.New()
	r.GET("/sse", h.HandleSSE)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/sse", nil)
	r.ServeHTTP(w, req)
	require.Contains(t, w.Body.String(), "token is required")
}

func TestSSEHandler_joinLeaveAndNotify(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewSSEHandler(nil)
	user := &models.User{ID: 3, Username: "carol"}
	r := gin.New()
	r.POST("/join", withUser(user), h.JoinGroup)
	r.POST("/leave", withUser(user), h.LeaveGroup)
	r.POST("/notify", withUser(user), h.SendNotification)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/join?group_id=g1", nil)
	r.ServeHTTP(w, req)
	require.Contains(t, w.Body.String(), "Successfully joined")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/leave?group_id=g1", nil)
	r.ServeHTTP(w, req)
	require.Contains(t, w.Body.String(), "Successfully left")

	payload, _ := json.Marshal(map[string]any{"type": "info", "data": "hello"})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/notify", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Contains(t, w.Body.String(), "Notification sent")
}
