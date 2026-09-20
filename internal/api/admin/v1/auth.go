package v1

import (
	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/pkg/captcha"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// GetCaptcha generates and returns a captcha image.
func (h *AuthHandler) GetCaptcha(c *gin.Context) {
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

// Login handles user authentication and returns a JWT token.
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	if !captcha.VerifyCaptcha(req.CaptchaID, req.CaptchaCode) {
		response.Error(c, response.CodeInvalidCaptcha, "invalid captcha")
		return
	}

	resp, err := h.auth.Login(c.Request.Context(), &req)
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

// RefreshToken handles token refresh requests.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		response.UnauthorizedError(c)
		return
	}
	userModel := user.(*models.User)
	token, err := h.auth.RefreshToken(c.Request.Context(), userModel.ID)
	if err != nil {
		response.Error(c, response.CodeServerError, "failed to refresh token")
		return
	}
	response.Success(c, gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   h.auth.GetConfig().JWT.ExpireTime,
	})
}

// Logout handles user logout requests.
func (h *AuthHandler) Logout(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		response.UnauthorizedError(c)
		return
	}
	userModel := user.(*models.User)
	_ = h.auth.Logout(c.Request.Context(), userModel.ID)
	response.Success(c, gin.H{"message": "logged out successfully"})
}
