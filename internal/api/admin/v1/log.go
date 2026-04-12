package v1

import (
	"strconv"

	"app/internal/core/services"
	"app/pkg/ginext"
	"app/pkg/response"

	"github.com/gin-gonic/gin"
)

// ListLoginLogs returns a list of login logs
func ListLoginLogs(c *gin.Context) {
	// Get query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// Get log service
	logSvc, ok := ginext.GetService[*services.LogService](c, "logService")
	if !ok {
		return
	}

	// Get logs with pagination
	logs, total, err := logSvc.GetLoginLogs(c.Request.Context(), page, pageSize)
	if err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	response.PageSuccess(c, logs, total, page, pageSize)
}

// ListOperationLogs returns a list of operation logs
func ListOperationLogs(c *gin.Context) {
	// Get query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// Get log service
	logSvc, ok := ginext.GetService[*services.LogService](c, "logService")
	if !ok {
		return
	}

	// Get logs with pagination
	logs, total, err := logSvc.GetOperationLogs(c.Request.Context(), page, pageSize)
	if err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	response.PageSuccess(c, logs, total, page, pageSize)
}

// GetUserLogs returns a user's login and operation logs
func GetUserLogs(c *gin.Context) {
	// Get user ID from path parameter
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "Invalid user ID")
		return
	}

	// Get log service
	logSvc, ok := ginext.GetService[*services.LogService](c, "logService")
	if !ok {
		return
	}

	// Get user's logs
	loginLogs, err := logSvc.GetUserLoginLogs(c.Request.Context(), uint(userID))
	if err != nil {
		response.ServerError(c)
		return
	}

	operationLogs, err := logSvc.GetUserOperationLogs(c.Request.Context(), uint(userID))
	if err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, gin.H{
		"login_logs":     loginLogs,
		"operation_logs": operationLogs,
	})
}
