package handlers

import (
	"strconv"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

func clientIDFromUser(user *models.User) string {
	if user == nil {
		return ""
	}
	if user.Username != "" {
		return user.Username
	}
	return strconv.FormatUint(uint64(user.ID), 10)
}

// currentUserID returns the realtime client id from JWT middleware context.
func currentUserID(c *gin.Context) (string, bool) {
	v, ok := c.Get("user")
	if !ok || v == nil {
		response.UnauthorizedError(c)
		return "", false
	}
	user, ok := v.(*models.User)
	if !ok || user == nil {
		response.UnauthorizedError(c)
		return "", false
	}
	id := clientIDFromUser(user)
	if id == "" {
		response.UnauthorizedError(c)
		return "", false
	}
	return id, true
}

func requireGroupID(c *gin.Context) (string, bool) {
	groupID := c.Query("group_id")
	if groupID == "" {
		response.ParamError(c, "group_id is required")
		return "", false
	}
	return groupID, true
}
