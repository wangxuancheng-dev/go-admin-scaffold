package v1

import (
	"strconv"

	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

// LogHandler handles log list endpoints.
type LogHandler struct {
	logs *services.LogService
}

func NewLogHandler(logs *services.LogService) *LogHandler {
	return &LogHandler{logs: logs}
}

func (h *LogHandler) ListLoginLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	logs, total, err := h.logs.GetLoginLogs(c.Request.Context(), page, pageSize)
	if err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	response.PageSuccess(c, logs, total, page, pageSize)
}

func (h *LogHandler) ListOperationLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	logs, total, err := h.logs.GetOperationLogs(c.Request.Context(), page, pageSize)
	if err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	response.PageSuccess(c, logs, total, page, pageSize)
}
