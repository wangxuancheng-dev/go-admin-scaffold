package handlers

import (
	"go-admin-scaffold/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
)

// TestHandler 测试处理器
type TestHandler struct{}

// NewTestHandler 创建测试处理器
func NewTestHandler() *TestHandler {
	return &TestHandler{}
}

// RateLimitTest 限流测试接口
func (h *TestHandler) RateLimitTest(c *gin.Context) {
	// 模拟一些处理时间
	time.Sleep(100 * time.Millisecond)

	response.Success(c, gin.H{
		"message": "Rate limit test successful",
		"time":    time.Now().Format(time.RFC3339),
	})
}
