package v1

import (
	"app/internal/core/models"
	"app/internal/core/services"
	"app/pkg/captcha"
	"app/pkg/ginext"
	"app/pkg/response"

	"github.com/gin-gonic/gin"
)

// GetCaptcha generates and returns a captcha image
func GetCaptcha(c *gin.Context) {
	id, b64s, err := captcha.GenerateCaptcha()
	if err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, gin.H{
		"captcha_id":    id,
		"captcha_image": b64s,
	})
}

// Login handles user authentication and returns a JWT token
func Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	// Verify captcha
	if !captcha.VerifyCaptcha(req.CaptchaID, req.CaptchaCode) {
		response.Error(c, response.CodeInvalidCaptcha, "invalid captcha")
		return
	}

	authSvc, ok := ginext.GetService[*services.AuthService](c, "authService")
	if !ok {
		return
	}
	resp, err := authSvc.Login(c.Request.Context(), &req)
	if err != nil {
		if err == services.ErrInvalidCredentials {
			response.Error(c, response.CodeInvalidCredentials, "invalid credentials")
			return
		}
		if err == services.ErrUserInactive {
			response.Error(c, response.CodeForbidden, "user is inactive")
			return
		}
		response.ServerError(c)
		return
	}

	response.Success(c, resp)
}

// RefreshToken handles token refresh requests
func RefreshToken(c *gin.Context) {
	// Get user from context (set by JWT middleware)
	user, exists := c.Get("user")
	if !exists {
		response.UnauthorizedError(c)
		return
	}

	userModel := user.(*models.User)
	authSvc, ok := ginext.GetService[*services.AuthService](c, "authService")
	if !ok {
		return
	}
	token, err := authSvc.RefreshToken(c.Request.Context(), userModel.ID)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to refresh token")
		return
	}

	response.Success(c, gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   authSvc.GetConfig().JWT.ExpireTime,
	})
}

// Logout handles user logout requests
func Logout(c *gin.Context) {
	// Get user from context (set by JWT middleware)
	user, exists := c.Get("user")
	if !exists {
		response.UnauthorizedError(c)
		return
	}

	userModel := user.(*models.User)
	authSvc, ok := ginext.GetService[*services.AuthService](c, "authService")
	if !ok {
		return
	}

	// Log the logout action
	err := authSvc.Logout(c.Request.Context(), userModel.ID)
	if err != nil {
		// Even if logout logging fails, we still consider logout successful
		// as the client-side token will be removed
		response.Success(c, gin.H{"message": "logged out successfully"})
		return
	}

	response.Success(c, gin.H{"message": "logged out successfully"})
}
