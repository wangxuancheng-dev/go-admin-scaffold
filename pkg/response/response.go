package response

import (
	"net/http"

	"app/pkg/utils"

	"github.com/gin-gonic/gin"
)

// Response represents the unified response structure
type Response struct {
	Code    int         `json:"code"`     // Business status code
	Message string      `json:"message"`  // Response message
	Data    interface{} `json:"data"`     // Response data
	TraceID string      `json:"trace_id"` // Trace ID for request tracking
}

// PaginationMeta matches the pagination object in list responses.
type PaginationMeta struct {
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// PagedList is the standard list payload returned by PageSuccess (items + pagination).
type PagedList struct {
	Items      interface{}    `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

// Response codes
const (
	CodeSuccess            = 0     // Success
	CodeParamError         = 10000 // Parameter error
	CodeValidationError    = 10001 // Validation error
	CodeServerError        = 10002 // Server error
	CodeNotFound           = 10003 // Not found
	CodeBusinessError      = 10004 // Business error
	CodeUnauthorized       = 10005 // Unauthorized
	CodeForbidden          = 10006 // Forbidden
	CodeCaptchaError       = 10007 // Captcha error
	CodeInvalidCaptcha     = 10008 // Invalid captcha
	CodeInvalidCredentials = 10009 // Invalid credentials
	CodeEmailTaken         = 10010 // Email already taken
	CodePermissionDenied   = 10011 // Permission denied
	CodeTooManyRequests    = 10012 // Rate limited
)

// Success sends a successful response
func Success(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, CodeSuccess, "success", data)
}

// Error sends an error response with trace ID. HTTP status is always 200; clients must use json.code.
func Error(c *gin.Context, code int, message string) {
	resp := Response{
		Code:    code,
		Message: message,
		Data:    nil,
		TraceID: utils.GetTraceID(c),
	}
	c.JSON(http.StatusOK, resp)
}

// ValidationError sends a validation error response
func ValidationError(c *gin.Context, message string) {
	Error(c, CodeValidationError, message)
}

// NotFoundError sends a not found error response
func NotFoundError(c *gin.Context) {
	Error(c, CodeNotFound, "Resource not found")
}

// BusinessError sends a business error response
func BusinessError(c *gin.Context, message string) {
	Error(c, CodeBusinessError, message)
}

// PageSuccess sends a successful paginated response
func PageSuccess(c *gin.Context, items interface{}, total int64, page, pageSize int) {
	Success(c, PagedList{
		Items: items,
		Pagination: PaginationMeta{
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// JSON sends a JSON response with trace ID
func JSON(c *gin.Context, httpStatus, code int, message string, data interface{}) {
	resp := Response{
		Code:    code,
		Message: message,
		Data:    data,
		TraceID: utils.GetTraceID(c),
	}
	c.JSON(httpStatus, resp)
}

// ServerError returns CodeServerError (HTTP 200, body.code).
func ServerError(c *gin.Context) {
	Error(c, CodeServerError, "Internal server error")
}

// UnauthorizedError returns CodeUnauthorized (HTTP 200, body.code).
func UnauthorizedError(c *gin.Context) {
	Error(c, CodeUnauthorized, "Unauthorized")
}

// ForbiddenError returns CodeForbidden (HTTP 200, body.code).
func ForbiddenError(c *gin.Context) {
	Error(c, CodeForbidden, "Forbidden")
}

// Unauthorized sends CodeUnauthorized (HTTP 200, body.code).
func Unauthorized(c *gin.Context, message string) {
	Error(c, CodeUnauthorized, message)
}

// Forbidden sends CodeForbidden (HTTP 200, body.code).
func Forbidden(c *gin.Context, message string) {
	Error(c, CodeForbidden, message)
}

// NotFound sends CodeNotFound (HTTP 200, body.code).
func NotFound(c *gin.Context, message string) {
	Error(c, CodeNotFound, message)
}

// ParamError sends CodeParamError (HTTP 200, body.code).
func ParamError(c *gin.Context, message string) {
	Error(c, CodeParamError, message)
}

// TooManyRequests sends CodeTooManyRequests (HTTP 200, body.code).
func TooManyRequests(c *gin.Context, message string) {
	if message == "" {
		message = "Too many requests"
	}
	Error(c, CodeTooManyRequests, message)
}
