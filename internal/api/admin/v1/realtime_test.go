package v1_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	adminv1 "go-admin-scaffold/internal/api/admin/v1"
	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/services"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRealtimeHandler_IssueTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	tickets := services.NewRealtimeTicketService(rdb, time.Minute)
	h := adminv1.NewRealtimeHandler(tickets)
	r := gin.New()
	r.POST("/ticket", func(c *gin.Context) {
		c.Set("user", &models.User{ID: 9, Username: "u"})
		h.IssueTicket(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/ticket", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].(map[string]interface{})
	require.NotEmpty(t, data["ticket"])
}
